package main

import (
	"log"
	"os"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
	"github.com/paypal/gatt/examples/service"
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

func (a *adaptadorBluetooth) servicoPrincipal() *gatt.Service {
	s := gatt.NewService(gatt.MustParseUUID("19fc95c0-c111-11e3-9904-0002a5d5c51b"))
	s.AddCharacteristic(gatt.MustParseUUID("45fac9e0-c111-11e3-9246-0002a5d5c51b")).HandleWriteFunc(
		func(r gatt.Request, data []byte) (status byte) {
			a.processar(data)
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
	a.device.Handle(
		gatt.CentralConnected(a.conexao()),
		gatt.CentralDisconnected(a.desconexao()),
	)
	onStateChanged := func(d gatt.Device, s gatt.State) {
		a.registrador.Printf("Estado: %s\n", s)
		switch s {
		case gatt.StatePoweredOn:
			d.AddService(service.NewGapService("Solutech Home Connect 1"))
			d.AddService(service.NewGattService())
			s1 := a.servicoPrincipal()
			d.AddService(s1)
			d.AdvertiseNameAndServices("Solutech Home Connect 1", []gatt.UUID{s1.UUID()})
			d.AdvertiseIBeacon(gatt.MustParseUUID("AA6062F098CA42118EC4193EB73CCEB6"), 1, 2, -59)
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

func (a *adaptadorBluetooth) processar(dados []byte) (r *requisicao) {
	//TODO: especificar e implementar protocolo de comunicação por bluetooth
	s := string(dados)
	a.registrador.Printf("%d bytes recebidos\n", len(dados))
	a.registrador.Printf("Conteúdo: %s\n", s)
	return
}
