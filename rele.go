package main

import (
	"log"
	"time"

	rpio "github.com/stianeikeland/go-rpio"
)

type rele struct {
	pino    int
	comando string
}

func (r *rele) acionar() {
	if err := rpio.Open(); err != nil {
		data := time.Now()
		dataTexto := data.Format("Monday 02-01-2006 15:04:05")
		log.Printf("%s: Erro ao acionar relé\n", dataTexto)
		log.Panicf("%s: %v", dataTexto, *r)
	}
	defer rpio.Close()
	pin := rpio.Pin(r.pino)
	pin.Output()
	switch r.comando {
	default:
		pin.Toggle()
	case "ligar":
		pin.High()
	case "desligar":
		pin.Low()
	}
}
