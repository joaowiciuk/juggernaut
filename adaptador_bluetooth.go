package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

type adaptadorBluetooth struct {
	registro    *os.File
	registrador *log.Logger
	device      gatt.Device
	descSSIDS   bool
	banco       *banco
}

func newAdaptadorBluetooth() (a *adaptadorBluetooth) {
	d, err := gatt.NewDevice(option.DefaultServerOptions...)
	if err != nil {
		log.Printf("Falha ao manipular dispositivo bluetooth, err: %s\n", err)
	}
	return &adaptadorBluetooth{
		device: d,
	}
}

func (a *adaptadorBluetooth) conexao() (f func(gatt.Central)) {
	return func(c gatt.Central) {
		a.registrador.Printf("Novo dispositivo conectado com ID %s\n", c.ID())
	}
}

func (a *adaptadorBluetooth) desconexao() (f func(gatt.Central)) {
	return func(c gatt.Central) {
		a.registrador.Printf("%s desconectou-se.\n", c.ID())
	}
}

func (a *adaptadorBluetooth) servicoUnico() *gatt.Service {
	s := gatt.NewService(gatt.UUID16(0x1815))

	requisicao := false

	lerTemp := s.AddCharacteristic(gatt.MustParseUUID("aee5af4f-d1a8-4855-b770-b912519327d6"))
	lerTemp.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		pendente := true
		for pendente {
			a.registrador.Printf("Iniciando leitura de temperatura...")
			cmd := exec.Command("/bin/sh", "-c", "vcgencmd measure_temp")
			//Saída padrão do comando
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				a.registrador.Println(err)
				break
			}
			//Inicia o comando porém não aguarda finalização
			if err := cmd.Start(); err != nil {
				a.registrador.Println(err)
				break
			}
			//Converte a saída do comando para string
			buf := new(bytes.Buffer)
			buf.ReadFrom(stdout)
			output := buf.String()
			//Aguarda até que o comando finalize
			if err := cmd.Wait(); err != nil {
				a.registrador.Println(err)
				break
			}
			//Filtra a saída do comando
			re := regexp.MustCompile(`temp=(.*)'C`)
			submatches := re.FindAllStringSubmatch(output, -1)
			temp, err := strconv.ParseFloat(submatches[0][1], 64)
			if err != nil {
				a.registrador.Println(err)
				break
			}
			a.registrador.Printf("Temperatura lida %.2f\n", temp)
			fmt.Fprintf(rsp, "%f", temp)
			pendente = false
		}
		//Intervalo para não estressar o dispositivo
		time.Sleep(time.Second * 1)
	})

	solicitarSSIDs := s.AddCharacteristic(gatt.MustParseUUID("351e784a-4099-405e-8031-e4b473e668a4"))
	solicitarSSIDs.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		if len(data) == 1 && data[0] == 0x79 {
			a.registrador.Printf("Descoberta de SSIDs solicitada pelo cliente GATT")
			requisicao = true
		} else {
			a.registrador.Printf("Descoberta de SSIDs recusada pelo cliente GATT")
			requisicao = false
		}
		return gatt.StatusSuccess
	})

	notificarSSIDs := s.AddCharacteristic(gatt.MustParseUUID("34a97fc8-5118-4484-b022-0c8a467cd533"))
	notificarSSIDs.HandleNotifyFunc(func(r gatt.Request, notifier gatt.Notifier) {
		//Enquanto as notificações não forem desativadas para a Characteristic...
		for !notifier.Done() {
			//Repete tentativa de descoberta de SSIDs pelo servidor GATT
			for requisicao {
				a.registrador.Printf("Iniciando tentativa de descoberta de SSIDs pelo pelo servidor GATT...")

				//Comando para verificar redes wifi disponíveis
				cmd := exec.Command("/bin/sh", "-c", "sudo iw dev wlan0 scan | grep SSID")

				//Saída padrão do comando
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					a.registrador.Println(err)
					break
				}

				//Inicia o comando porém não aguarda finalização
				if err := cmd.Start(); err != nil {
					a.registrador.Println(err)
					break
				}

				//Converte a saída do comando para string
				buf := new(bytes.Buffer)
				buf.ReadFrom(stdout)
				output := buf.String()

				//Aguarda até que o comando finalize
				if err := cmd.Wait(); err != nil {
					a.registrador.Println(err)
					break
				}

				//Filtra a saída do comando
				re := regexp.MustCompile(`\ *SSID:\ (.*)`)
				submatches := re.FindAllStringSubmatch(output, -1)
				type SSID struct {
					Lista []string `json:"ssids"`
				}
				ssid := SSID{
					Lista: make([]string, 0),
				}
				for _, submatch := range submatches {
					ssid.Lista = append(ssid.Lista, submatch[1])
				}

				//Nenhum SSID encontrado
				if len(ssid.Lista) == 0 {
					a.registrador.Printf("Nenhum SSID encontrado.\n")
					a.registrador.Printf("Descoberta de SSIDs falhou.\n")
					requisicao = false
					return
				}

				//Converte os SSIDs para uma string JSON codificada em base 64
				src, err := json.Marshal(ssid)
				if err != nil {
					a.registrador.Printf("Falha ao codificar SSIDs em base 64.\n")
					requisicao = false
					return
				}
				size := ((4 * len(src) / 3) + 3) & ^3
				dst := make([]byte, size)
				base64.StdEncoding.Encode(dst, src)
				reader := bytes.NewReader(dst)

				//Registra todos os ssids encontrados
				for _, s := range ssid.Lista {
					a.registrador.Printf("%s\n", s)
				}

				//Buffer de transferência para enviar em pedaços
				transf := make([]byte, 8)

				//Inicia a transferência de ssidSource por mensagens do notifier
				// >> IMPORTANTE: para esta característica são permitidos apenas 8 bytes por mensagem <<
				for {
					k, err := reader.Read(transf)
					if err == io.EOF {
						a.registrador.Printf("Descoberta de SSIDs encerrada com sucesso.")
						requisicao = false
						break
					}

					//registra o buffer de transferência
					a.registrador.Printf("transf[:%d] = %q\n", k, transf[:k])

					//envia o buffer de transferência pelo notifier
					fmt.Fprintf(notifier, "%s", transf[:k])
				}
			}
			//Intervalo para não estressar o dispositivo
			time.Sleep(time.Second * 1)
		}
	})

	lerAmbiente := s.AddCharacteristic(gatt.MustParseUUID("ff39ae7e-61b6-4f67-af74-324e7af948bd"))
	lerAmbiente.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		ambiente := a.banco.lerAmbiente()
		rsp.SetStatus(gatt.StatusSuccess)
		fmt.Fprintf(rsp, "%s", ambiente)
	})

	escreverAmbiente := s.AddCharacteristic(gatt.MustParseUUID("2f54b94a-a6fe-4d5f-a4ca-932a362eba10"))
	escreverAmbiente.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		ambiente := string(data)
		a.banco.salvarAmbiente(ambiente)
		return gatt.StatusSuccess
	})

	lerIP := s.AddCharacteristic(gatt.MustParseUUID("02e9a221-8643-451e-ad92-deeec489c44b"))
	lerIP.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		ip := a.banco.lerIP()
		rsp.SetStatus(gatt.StatusSuccess)
		fmt.Fprintf(rsp, "%s", ip)
		a.registrador.Printf("IP solicitado: %s\n", ip)
	})

	escreverIP := s.AddCharacteristic(gatt.MustParseUUID("92e6b940-1ed5-43fb-b942-6ac51ad5d72d"))
	escreverIP.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		ip := string(data)
		a.banco.salvarIP(ip)
		a.registrador.Printf("IP escrito: %s\n", ip)
		return gatt.StatusSuccess
	})

	escreverNomeCliente := s.AddCharacteristic(gatt.MustParseUUID("cde083d8-b20c-4709-b756-2f219a911994"))
	escreverNomeCliente.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		if a.banco.lerNomeCliente() == "" {
			nomeCliente := string(data)
			a.banco.salvarNomeCliente(nomeCliente)
			a.registrador.Printf("nome-cliente escrito: %s\n", nomeCliente)
		}
		return gatt.StatusSuccess
	})

	lerNomeCliente := s.AddCharacteristic(gatt.MustParseUUID("55cc9c0d-d42d-4f0f-850c-00b1809007e7"))
	lerNomeCliente.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		nomeCliente := a.banco.lerNomeCliente()
		if nomeCliente == "" {
			fmt.Fprint(rsp, "indefinido")
		} else {
			fmt.Fprintf(rsp, "%s", nomeCliente)
		}
		a.registrador.Printf("nome-cliente solicitado: %s\n", nomeCliente)
	})

	return s
}

func (a *adaptadorBluetooth) servTemperatura() *gatt.Service {
	solicitada := false
	s := gatt.NewService(gatt.UUID16(0x1815))
	caracEnvTemp := s.AddCharacteristic(gatt.MustParseUUID("aee5af4f-d1a8-4855-b770-b912519327d6"))
	caracEnvTemp.HandleNotifyFunc(func(r gatt.Request, notifier gatt.Notifier) {

		//Enquanto as notificações não forem desativadas para a Characteristic...
		for !notifier.Done() {

			for solicitada {
				a.registrador.Printf("Iniciando leitura de temperatura...")

				cmd := exec.Command("/bin/sh", "-c", "vcgencmd measure_temp")

				//Saída padrão do comando
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					a.registrador.Println(err)
					break
				}

				//Inicia o comando porém não aguarda finalização
				if err := cmd.Start(); err != nil {
					a.registrador.Println(err)
					break
				}

				//Converte a saída do comando para string
				buf := new(bytes.Buffer)
				buf.ReadFrom(stdout)
				output := buf.String()

				//Aguarda até que o comando finalize
				if err := cmd.Wait(); err != nil {
					a.registrador.Println(err)
					break
				}

				//Filtra a saída do comando
				re := regexp.MustCompile(`temp=(.*)'C`)
				submatches := re.FindAllStringSubmatch(output, -1)
				value, err := strconv.ParseFloat(submatches[0][1], 64)
				if err != nil {
					a.registrador.Println(err)
					break
				}
				type Temp struct {
					Value float64 `json:"temperatura"`
				}
				temp := Temp{
					Value: value,
				}

				src, err := json.Marshal(temp)
				if err != nil {
					a.registrador.Printf("Falha ao codificar temperatura em base 64.\n")
					a.descSSIDS = false
					return
				}
				size := ((4 * len(src) / 3) + 3) & ^3
				dst := make([]byte, size)
				base64.StdEncoding.Encode(dst, src)
				reader := bytes.NewReader(dst)

				//Registra a temperatura
				a.registrador.Printf("%.2f\n", temp.Value)

				//Buffer de transferência para enviar em pedaços
				transf := make([]byte, 8)

				//Inicia a transferência de ssidSource por mensagens do notifier
				// >> IMPORTANTE: para esta característica são permitidos apenas 8 bytes por mensagem <<
				for {
					k, err := reader.Read(transf)
					if err == io.EOF {
						a.registrador.Printf("Leitura de temperatura encerrada com sucesso.")
						solicitada = false
						break
					}

					//registra o buffer de transferência
					a.registrador.Printf("transf[:%d] = %q\n", k, transf[:k])

					//envia o buffer de transferência pelo notifier
					fmt.Fprintf(notifier, "%s", transf[:k])
				}
			}

			//Intervalo para não estressar o dispositivo
			time.Sleep(time.Second * 1)
		}
	})

	caracSolTemp := s.AddCharacteristic(gatt.MustParseUUID("51aafba2-2d8b-48de-84a1-1d5746af5447"))
	caracSolTemp.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		if len(data) == 1 && data[0] == 0x79 {
			a.registrador.Printf("Leitura de temperatura solicitada pelo cliente GATT")
			solicitada = true
		} else {
			a.registrador.Printf("Leitura de temperatura solicitada pelo cliente GATT")
			solicitada = false
		}
		return gatt.StatusSuccess
	})

	return s
}

func (a *adaptadorBluetooth) servWifi() *gatt.Service {
	s := gatt.NewService(gatt.UUID16(0x1815))
	caracSSIDs := s.AddCharacteristic(gatt.UUID16(0x2A04))
	caracSSIDs.HandleNotifyFunc(func(r gatt.Request, notifier gatt.Notifier) {

		//Enquanto as notificações não forem desativadas para a Characteristic...
		for !notifier.Done() {

			//Repete tentativa de descoberta de SSIDs pelo servidor GATT
			for a.descSSIDS {
				a.registrador.Printf("Iniciando tentativa de descoberta de SSIDs pelo pelo servidor GATT...")

				//Comando para verificar redes wifi disponíveis
				cmd := exec.Command("/bin/sh", "-c", "sudo iw dev wlan0 scan | grep SSID")

				//Saída padrão do comando
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					a.registrador.Println(err)
					break
				}

				//Inicia o comando porém não aguarda finalização
				if err := cmd.Start(); err != nil {
					a.registrador.Println(err)
					break
				}

				//Converte a saída do comando para string
				buf := new(bytes.Buffer)
				buf.ReadFrom(stdout)
				output := buf.String()

				//Aguarda até que o comando finalize
				if err := cmd.Wait(); err != nil {
					a.registrador.Println(err)
					break
				}

				//Filtra a saída do comando
				re := regexp.MustCompile(`\ *SSID:\ (.*)`)
				submatches := re.FindAllStringSubmatch(output, -1)
				type SSID struct {
					Lista []string `json:"ssids"`
				}
				ssid := SSID{
					Lista: make([]string, 0),
				}
				for _, submatch := range submatches {
					ssid.Lista = append(ssid.Lista, submatch[1])
				}

				//Nenhum SSID encontrado
				if len(ssid.Lista) == 0 {
					a.registrador.Printf("Nenhum SSID encontrado.\n")
					a.registrador.Printf("Descoberta de SSIDs falhou.\n")
					a.descSSIDS = false
					return
				}

				//Converte os SSIDs para uma string JSON codificada em base 64
				src, err := json.Marshal(ssid)
				if err != nil {
					a.registrador.Printf("Falha ao codificar SSIDs em base 64.\n")
					a.descSSIDS = false
					return
				}
				size := ((4 * len(src) / 3) + 3) & ^3
				dst := make([]byte, size)
				base64.StdEncoding.Encode(dst, src)
				reader := bytes.NewReader(dst)

				//Registra todos os ssids encontrados
				for _, s := range ssid.Lista {
					a.registrador.Printf("%s\n", s)
				}

				//Buffer de transferência para enviar em pedaços
				transf := make([]byte, 8)

				//Inicia a transferência de ssidSource por mensagens do notifier
				// >> IMPORTANTE: para esta característica são permitidos apenas 8 bytes por mensagem <<
				for {
					k, err := reader.Read(transf)
					if err == io.EOF {
						a.registrador.Printf("Descoberta de SSIDs encerrada com sucesso.")
						a.descSSIDS = false
						break
					}

					//registra o buffer de transferência
					a.registrador.Printf("transf[:%d] = %q\n", k, transf[:k])

					//envia o buffer de transferência pelo notifier
					fmt.Fprintf(notifier, "%s", transf[:k])
				}
			}

			//Intervalo para não estressar o dispositivo
			time.Sleep(time.Second * 1)
		}
	})

	caracSolicitarDesc := s.AddCharacteristic(gatt.MustParseUUID("351e784a-4099-405e-8031-e4b473e668a4"))
	caracSolicitarDesc.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		if len(data) == 1 && data[0] == 0x79 {
			a.registrador.Printf("Descoberta de SSIDs solicitada pelo cliente GATT")
			a.descSSIDS = true
		} else {
			a.registrador.Printf("Descoberta de SSIDs recusada pelo cliente GATT")
			a.descSSIDS = false
		}
		return gatt.StatusSuccess
	})

	return s
}

func (a *adaptadorBluetooth) servAmbiente() *gatt.Service {
	s := gatt.NewService(gatt.UUID16(0x1815))
	caracObterAmbiente := s.AddCharacteristic(gatt.MustParseUUID("02e9a221-8643-451e-ad92-deeec489c44b"))
	caracObterAmbiente.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		ambiente := a.banco.lerAmbiente()
		rsp.SetStatus(gatt.StatusSuccess)
		fmt.Fprintf(rsp, "%s", ambiente)
	})

	caracDefinirAmbiente := s.AddCharacteristic(gatt.MustParseUUID("92e6b940-1ed5-43fb-b942-6ac51ad5d72d"))
	caracDefinirAmbiente.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		ambiente := string(data)
		a.banco.salvarAmbiente(ambiente)
		return gatt.StatusSuccess
	})

	return s
}

func (a *adaptadorBluetooth) servIP() *gatt.Service {
	s := gatt.NewService(gatt.UUID16(0x1815))
	caracObterIP := s.AddCharacteristic(gatt.MustParseUUID("02e9a221-8643-451e-ad92-deeec489c44b"))
	caracObterIP.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		ip := a.banco.lerIP()
		rsp.SetStatus(gatt.StatusSuccess)
		fmt.Fprintf(rsp, "%s", ip)
		a.registrador.Printf("IP solicitado: %s\n", ip)
	})

	caracDefinirIP := s.AddCharacteristic(gatt.MustParseUUID("92e6b940-1ed5-43fb-b942-6ac51ad5d72d"))
	caracDefinirIP.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		ip := string(data)
		a.banco.salvarIP(ip)
		a.registrador.Printf("IP escrito: %s\n", ip)
		return gatt.StatusSuccess
	})

	return s
}

func (a *adaptadorBluetooth) inicializar(endereco string, banco *banco) error {
	f, err := os.OpenFile(endereco, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	a.registro = f
	a.registrador = log.New(a.registro, "", log.Ldate|log.Ltime)
	if a.device == nil {
		a.registrador.Fatalf("erro: adaptador não consegue comunicar com dispositivo")
	}
	a.device.Handle(
		gatt.CentralConnected(a.conexao()),
		gatt.CentralDisconnected(a.desconexao()),
	)
	onStateChanged := func(d gatt.Device, s gatt.State) {
		a.registrador.Printf("Estado: %s\n", s)
		switch s {
		case gatt.StatePoweredOn:
			/* sWifi := a.servWifi()
			d.AddService(sWifi)
			sAmb := a.servAmbiente()
			d.AddService(sAmb)
			sTemp := a.servTemperatura()
			d.AddService(sTemp)
			sIp := a.servIP()
			d.AddService(sIp)
			d.AdvertiseNameAndServices("Solutech Home Connect", []gatt.UUID{sWifi.UUID(), sAmb.UUID(), sTemp.UUID(), sIp.UUID()}) */

			sUnico := a.servicoUnico()
			d.AddService(sUnico)
			d.AdvertiseNameAndServices("Solutech Home Connect", []gatt.UUID{sUnico.UUID()})
		default:
		}
	}
	a.registrador.Printf("Inicializando adaptador bluetooth...\n")
	a.device.Init(onStateChanged)
	a.banco = banco
	return nil
}

func (a *adaptadorBluetooth) finalizar() {
	a.registrador.Printf("Finalizando adaptador bluetooth...\n")
	a.registro.Close()
}
