package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	rpio "github.com/stianeikeland/go-rpio"
)

type RelayManager struct {
	LogFile *os.File
	Logger  *log.Logger
}

const (
	RelayToggle = "toggle"
	RelayOn     = "on"
	RelayOff    = "off"
)

type Relay struct {
	Pin     int    `json:"pin"`
	Command string `json:"command"`
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

func (rm *RelayManager) Close() {
	rm.LogFile.Close()
}

func (rm *RelayManager) Operate(relay Relay) {
	if err := rpio.Open(); err != nil {
		rm.Logger.Printf("opening rpio: %v\n", err)
		return
	}
	defer rpio.Close()
	rpioPin := rpio.Pin(relay.Pin)
	rpioPin.Output()
	switch relay.Command {
	case RelayToggle:
		rpioPin.Toggle()
		rm.Logger.Printf("Toggle relay on pin %d.\n", relay.Pin)
	case RelayOn:
		rpioPin.Low()
		rm.Logger.Printf("Switch on relay on pin %d.\n", relay.Pin)
	case RelayOff:
		rpioPin.High()
		rm.Logger.Printf("Switch off relay on pin %d.\n", relay.Pin)
	default:
		rpioPin.Toggle()
		rm.Logger.Printf("Toggle relay on pin %d.\n", relay.Pin)
	}

	return
}

func (rm *RelayManager) RelayHandler(w http.ResponseWriter, r *http.Request) {
	rm.Logger.Printf("relay handler: request received.\n")
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		rm.Logger.Printf("relay handler: %v", err)
		return
	}
	defer conn.Close()
	for {
		var relay Relay
		err := conn.ReadJSON(&relay)
		rm.Logger.Printf("relay handler: relay received: %v.\n", relay)
		if err != nil {
			rm.Logger.Printf("relay handler: %v\n", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		rm.Operate(relay)
	}
}

func (rm *RelayManager) NoWebSocketRelayHandler(w http.ResponseWriter, r *http.Request) {
	pinStr := mux.Vars(r)["pin"]
	pin, _ := strconv.Atoi(pinStr)
	relay := Relay{
		Pin:     pin,
		Command: RelayToggle,
	}
	rm.Logger.Printf("NoWebSocketRelayHandler: relay received: %v.\n", relay)
	rm.Operate(relay)
	w.WriteHeader(http.StatusOK)
}

func (rm *RelayManager) NoWebSocketInfraredHandler(w http.ResponseWriter, r *http.Request) {
	pinStr := mux.Vars(r)["pin"]
	pin, _ := strconv.Atoi(pinStr)
	relay := Relay{
		Pin:     pin,
		Command: RelayToggle,
	}
	rm.Logger.Printf("NoWebSocketRelayHandler: relay received: %v.\n", relay)
	rm.Operate(relay)
	w.WriteHeader(http.StatusOK)
}
