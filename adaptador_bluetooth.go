package main

import (
	"bytes"
	"fmt"
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
	s := gatt.NewService(gatt.MustParseUUID("ac044f25-921b-4a9a-acaa-64c9fb77982a"))
	c := s.AddCharacteristic(gatt.MustParseUUID("87a040df-b13f-46d3-be03-ade57dcf1f07"))
	c.HandleNotifyFunc(
		func(r gatt.Request, n gatt.Notifier) {
			for !n.Done() {
				cmd := exec.Command("/bin/sh", "-c", "sudo iw dev wlan0 scan | grep SSID")
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					a.registrador.Println(err)
					return
				}
				if err := cmd.Start(); err != nil {
					a.registrador.Println(err)
					return
				}
				buf := new(bytes.Buffer)
				buf.ReadFrom(stdout)
				output := buf.String()
				if err := cmd.Wait(); err != nil {
					a.registrador.Println(err)
					return
				}
				re := regexp.MustCompile(`\ *SSID:\ (.*)`)
				submatches := re.FindAllStringSubmatch(output, -1)
				ssids := make([]string, 0)
				for _, submatch := range submatches {
					ssids = append(ssids, submatch[1])
				}
				if len(ssids) < 2 {
					a.registrador.Printf("error: no ssid")
					return
				}
				for _, ssid := range ssids {
					a.registrador.Printf("%s\n", ssid)
					fmt.Fprintf(n, "%s", ssid)
				}
				time.Sleep(time.Second * 10)
			}
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
