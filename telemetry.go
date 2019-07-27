package main

import (
	"bytes"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type TelemetryManager struct {
	DatabaseManager *DatabaseManager
	LogFile         *os.File
	Logger          *log.Logger
	Websocket       *websocket.Conn
}

func NewTelemetryManager() TelemetryManager {
	return TelemetryManager{}
}

func (t *TelemetryManager) Initialize(logPath string, database *DatabaseManager) (err error) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	t.LogFile = f
	t.Logger = log.New(t.LogFile, "", log.Ldate|log.Ltime)
	t.DatabaseManager = database
	t.Logger.Printf("TelemetryManager started.\n")
	go t.Communicate()
	return nil
}

func (t *TelemetryManager) Finish() {
	t.Logger.Printf("TelemetryManager finished.\n")
	t.LogFile.Close()
	t.Websocket.Close()
}

func (t *TelemetryManager) ReadTemperature() float64 {
	cmd := exec.Command("/bin/sh", "-c", "vcgencmd measure_temp")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Logger.Println(err)
		return 0
	}
	if err := cmd.Start(); err != nil {
		t.Logger.Println(err)
		return 0
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(stdout)
	output := buf.String()
	if err := cmd.Wait(); err != nil {
		t.Logger.Println(err)
		return 0
	}
	re := regexp.MustCompile(`temp=(.*)'C`)
	submatches := re.FindAllStringSubmatch(output, -1)
	value, err := strconv.ParseFloat(submatches[0][1], 64)
	if err != nil {
		t.Logger.Println(err)
		return 0
	}
	return value
}

func (t *TelemetryManager) Communicate() {
	defer t.Communicate()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	environment := t.DatabaseManager.ReadEnvironment()
	var host string
	if environment == EnvironmentDevelopment {
		host = "179.234.70.32:8081"
	} else if environment == EnvironmentProduction {
		host = "http://solutech.site"
	}
	u := url.URL{Scheme: "ws", Host: host, Path: "/shc/telemetria"}
	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Logger.Printf("TelemetryManager#Communicate(): dial: %v\n", err)
		return
	}
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
	if t.DatabaseManager.ReadEnvironment() == EnvironmentDevelopment {
		ticker = time.NewTicker(5 * time.Second)
	} else if t.DatabaseManager.ReadEnvironment() == EnvironmentProduction {
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
			device := Device{
				UUID:        t.DatabaseManager.ReadUUID(),
				Identifier:  t.DatabaseManager.ReadIdentifier(),
				Temperature: t.ReadTemperature(),
				LastUpdate:  instant,
			}
			err = c.WriteJSON(device)
			if err != nil {
				t.Logger.Printf("writing JSON: %v\n", err)
				isUp = false
			}
		case <-interrupt:
			isUp = false
		}
	}
}
