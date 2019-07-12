package main

import (
	"bytes"
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

			//Se a descoberta de SSIDs não tiver sido solicitada, abortar
			if a.descSSIDS == false {
				return
			}

			//Comando para verificar redes wifi disponíveis
			cmd := exec.Command("/bin/sh", "-c", "sudo iw dev wlan0 scan | grep SSID")

			//Saída padrão do comando
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				a.registrador.Println(err)
				return
			}

			//Inicia o comando porém não aguarda finalização
			if err := cmd.Start(); err != nil {
				a.registrador.Println(err)
				return
			}

			//Converte a saída do comando para string
			buf := new(bytes.Buffer)
			buf.ReadFrom(stdout)
			output := buf.String()

			//Aguarda até que o comando finalize
			if err := cmd.Wait(); err != nil {
				a.registrador.Println(err)
				return
			}

			//Expressão regular para identificar a informação desejada na saída do comando
			re := regexp.MustCompile(`\ *SSID:\ (.*)`)
			submatches := re.FindAllStringSubmatch(output, -1)
			ssids := make([]string, 0)
			ssidsSource := new(bytes.Buffer)

			//Monta uma lista de ssids a partir da saída do comando
			//Também armazena essa lista no ssidsSource, para uso futuro
			for _, submatch := range submatches {
				ssids = append(ssids, submatch[1])
				io.WriteString(ssidsSource, submatch[1])
			}
			if len(ssids) < 2 {
				a.registrador.Printf("error: no ssid")
				return
			}

			//Registra todos os ssids encontrados
			for _, ssid := range ssids {
				a.registrador.Printf("%s\n", ssid)
			}

			//Buffer de transferência para enviar o ssidSource em pedaços de 8 bytes
			bufferTransf := make([]byte, 8)

			//Inicia a transferência de ssidSource por mensagens do notifier
			// >> IMPORTANTE: para esta característica são permitidos apenas 8 bytes por mensagem <<
			for {
				k, err := ssidsSource.Read(bufferTransf)

				//registra o buffer de transferência
				a.registrador.Printf("k = %v err = %v bufferTransf = %v\n", k, err, bufferTransf)

				//registra o buffer de transferência
				a.registrador.Printf("bufferTransf[:k] = %q\n", bufferTransf[:k])

				//envia o buffer de transferência pelo notifier
				fmt.Fprintf(notifier, "%s", bufferTransf[:k])
				if err == io.EOF {
					break
				}
			}

			//Aguarda 10 segundos até a próxima verificação, caso seja solicitada
			time.Sleep(time.Second * 10)
			a.descSSIDS = false
		}
	})

	caracSolicitarDesc := s.AddCharacteristic(gatt.MustParseUUID("351e784a-4099-405e-8031-e4b473e668a4"))
	caracSolicitarDesc.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		if string(data) == "sim" {
			a.descSSIDS = true
		}
		return gatt.StatusSuccess
	})

	return s
}

func (a *adaptadorBluetooth) inicializar(endereco string) error {
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
			d.AdvertiseNameAndServices("Solutech Home Connect", []gatt.UUID{descWifi.UUID()})
		default:
		}
	}
	a.registrador.Printf("Inicializando adaptador bluetooth...\n")
	a.device.Init(onStateChanged)
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
