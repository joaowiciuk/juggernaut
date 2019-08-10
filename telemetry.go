package main

import (
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

//	Responsibilities:
//	*	To send system state to the cloud
//	TelemetryManager
type TelemetryManager struct {
	*DeviceManager
	*DatabaseManager
	LogFile   *os.File
	Logger    *log.Logger
	Websocket *websocket.Conn
}

type TelemetryInfo struct {
	UUID        string    `json:"uuid"`
	Identifier  string    `json:"identifier"`
	Temperature float64   `json:"temperature"`
	LastUpdate  time.Time `json:"last_update"`
}

func NewTelemetryManager() TelemetryManager {
	return TelemetryManager{}
}

func (t *TelemetryManager) Initialize(logPath string, database *DatabaseManager, deviceManager *DeviceManager) (err error) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	t.LogFile = f
	t.Logger = log.New(t.LogFile, "", log.Ldate|log.Ltime)
	t.DatabaseManager = database
	t.DeviceManager = deviceManager
	t.Logger.Printf("TelemetryManager started.\n")
	return nil
}

func (t *TelemetryManager) Close() {
	t.Logger.Printf("TelemetryManager closed.\n")
	t.LogFile.Close()
	t.Websocket.Close()
}

func (t *TelemetryManager) Communicate() {
	t.Logger.Printf("TelemetryManager#Communicate(): Starting communication proccess...")
	defer t.Communicate()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	device := t.DatabaseManager.ReadDevice()
	environment := device.Info.Environment
	var host string
	if environment == EnvironmentDevelopment {
		host = "179.234.70.32:8081"
	} else if environment == EnvironmentProduction {
		host = "http://solutech.site"
	}
	u := url.URL{Scheme: "ws", Host: host, Path: "/shc/telemetria"}
	t.Logger.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Logger.Printf("communicating: %v\n", err)
		time.Sleep(15 * time.Second)
		return
	}
	t.Logger.Printf("TelemetryManager#Communicate(): Communication proccess started succesfuly.")
	defer c.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				t.Logger.Println("Ao receber mensagem: ", err)
				return
			}
			t.Logger.Printf("Recebido: %s", message)
		}
	}()
	t.Websocket = c
	var ticker *time.Ticker
	if environment == EnvironmentDevelopment {
		ticker = time.NewTicker(5 * time.Second)
	} else if environment == EnvironmentProduction {
		ticker = time.NewTicker(60 * time.Second)
	} else {
		ticker = time.NewTicker(15 * time.Second)
	}
	defer ticker.Stop()
	isUp := true
	for isUp {
		select {
		case <-done:
			isUp = false
		case instant := <-ticker.C:
			telemetryInfo := TelemetryInfo{
				Identifier:  device.Info.Identifier,
				LastUpdate:  instant,
				Temperature: t.DeviceManager.Temperature(),
				UUID:        device.Info.UUID,
			}
			err = c.WriteJSON(telemetryInfo)
			if err != nil {
				t.Logger.Printf("writing JSON: %v\n", err)
				isUp = false
			}
		case <-interrupt:
			isUp = false
		}
	}
}
