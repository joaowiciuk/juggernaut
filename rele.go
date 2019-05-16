package main

import (
	"log"

	rpio "github.com/stianeikeland/go-rpio"
)

type rele struct {
	pino    int
	comando string
}

func (r *rele) acionar() {
	if err := rpio.Open(); err != nil {
		log.Printf("Erro ao acionar relé\n")
		log.Panicf("%v", *r)
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
