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

func (a *adaptadorBluetooth) descobertaWifi() *gatt.Service {
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

					//registra o buffer de transferência
					a.registrador.Printf("transf[:%d] = %q\n", k, transf[:k])

					//envia o buffer de transferência pelo notifier
					fmt.Fprintf(notifier, "%s", transf[:k])
					if err == io.EOF {
						a.registrador.Printf("Descoberta de SSIDs encerrada com sucesso.")
						a.descSSIDS = false
						break
					}
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

func (a *adaptadorBluetooth) servicoConfigAmbiente() *gatt.Service {
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
			descWifi := a.descobertaWifi()
			d.AddService(descWifi)
			configAmb := a.servicoConfigAmbiente()
			d.AddService(configAmb)
			d.AdvertiseNameAndServices("Solutech Home Connect", []gatt.UUID{descWifi.UUID(), configAmb.UUID()})
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

//TODO: especificar e implementar protocolo de comunicação por bluetooth
func (a *adaptadorBluetooth) processar(dados []byte) (r *requisicao) {
	s := string(dados)
	a.registrador.Printf("%d bytes recebidos\n", len(dados))
	a.registrador.Printf("Conteúdo: %s\n", s)
	return
}
