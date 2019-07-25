package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"

	rpio "github.com/stianeikeland/go-rpio"
)

type RelayManager struct {
	LogFile *os.File
	Logger  *log.Logger
}

func NewRelayManager() *RelayManager {
	return &RelayManager{}
}

func (rm *RelayManager) Initialize(logPath string) (err error) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	rm.LogFile = f
	rm.Logger = log.New(f, "", log.Ldate|log.Ltime)
	rm.Logger.Printf("Relay manager started.\n")
	return nil
}

func (rm *RelayManager) Finish() {
	rm.LogFile.Close()
}

func (rm *RelayManager) Toggle(pin int) {
	if err := rpio.Open(); err != nil {
		rm.Logger.Printf("opening rpio: %v\n", err)
		return
	}
	defer rpio.Close()
	rpioPin := rpio.Pin(pin)
	rpioPin.Output()
	rpioPin.Toggle()
	rm.Logger.Printf("Pin %d toggled\n", pin)
}

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

func RelayHandlerToggler(rm *RelayManager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		pinStr := mux.Vars(r)["pin"]
		pin, _ := strconv.Atoi(pinStr)
		rm.Toggle(pin)
		w.WriteHeader(http.StatusOK)
	}
}
