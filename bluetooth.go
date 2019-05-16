package main

import (
	"fmt"
	"log"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
	"github.com/paypal/gatt/examples/service"
)

func adaptadorBluetooth() {
	d, err := gatt.NewDevice(option.DefaultServerOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s", err)
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
}

func aoConectar(c gatt.Central) {
	log.Printf("%s conectou-se.\n", c.ID())
}

func aoDesconectar(c gatt.Central) {
	log.Printf("%s desconectou-se.\n", c.ID())
}

func aproxDePI() *gatt.Service {
	s := gatt.NewService(gatt.MustParseUUID("19fc95c0-c111-11e3-9904-0002a5d5c51b"))
	s.AddCharacteristic(gatt.MustParseUUID("44fac9e0-c111-11e3-9246-0002a5d5c51b")).HandleReadFunc(
		func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
			log.Printf("Aproximação de PI solicitada.\n")
			fmt.Fprintf(rsp, "3.14159")
		})
	s.AddCharacteristic(gatt.MustParseUUID("45fac9e0-c111-11e3-9246-0002a5d5c51b")).HandleWriteFunc(
		func(r gatt.Request, data []byte) (status byte) {
			log.Printf("Requisição para escrita.\n")
			return gatt.StatusSuccess
		})
	return s
}
