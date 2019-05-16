package main

import (
	"fmt"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
	"github.com/paypal/gatt/examples/service"
)

func adaptadorBluetooth() {
	logger.Printf("Iniciando manipulador de dispositivo bluetooth\n")
	d, err := gatt.NewDevice(option.DefaultServerOptions...)
	if err != nil {
		logger.Printf("Falha ao manipular dispositivo bluetooth, err: %s\n", err)
	}
	d.Handle(
		gatt.CentralConnected(aoConectar),
		gatt.CentralDisconnected(aoDesconectar),
	)
	onStateChanged := func(d gatt.Device, s gatt.State) {
		fmt.Printf("State: %s\n", s)
		switch s {
		case gatt.StatePoweredOn:
			d.AddService(service.NewGapService("Solutech Home Connect 1"))
			d.AddService(service.NewGattService())
			s1 := aproxDePI()
			d.AddService(s1)
			d.AdvertiseNameAndServices("Solutech Home Connect 1", []gatt.UUID{s1.UUID()})
			d.AdvertiseIBeacon(gatt.MustParseUUID("AA6062F098CA42118EC4193EB73CCEB6"), 1, 2, -59)
		default:
		}
	}
	d.Init(onStateChanged)
	logger.Printf("Finalizando manipulador de dispositivo bluetooth\n")
}

func aoConectar(c gatt.Central) {
	logger.Printf("%s conectou-se.\n", c.ID())
}

func aoDesconectar(c gatt.Central) {
	logger.Printf("%s desconectou-se.\n", c.ID())
}

func aproxDePI() *gatt.Service {
	s := gatt.NewService(gatt.MustParseUUID("19fc95c0-c111-11e3-9904-0002a5d5c51b"))
	s.AddCharacteristic(gatt.MustParseUUID("44fac9e0-c111-11e3-9246-0002a5d5c51b")).HandleReadFunc(
		func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
			logger.Printf("Aproximação de PI solicitada.\n")
			fmt.Fprintf(rsp, "3.14159")
		})
	s.AddCharacteristic(gatt.MustParseUUID("45fac9e0-c111-11e3-9246-0002a5d5c51b")).HandleWriteFunc(
		func(r gatt.Request, data []byte) (status byte) {
			logger.Printf("Requisição para escrita\n")
			logger.Printf("$d bytes recebidos\n", len(data))
			return gatt.StatusSuccess
		})
	return s
}
