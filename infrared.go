package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/mux"
)

//	Responsibilities:
//	*	To handle infrared operation via Wifi - HTTP
//	InfraredManager
type InfraredManager struct {
	LogFile *os.File
	Logger  *log.Logger
}

func NewInfraredManager() *InfraredManager {
	return &InfraredManager{}
}

func (i *InfraredManager) Initialize(logPath string) (err error) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	i.LogFile = f
	i.Logger = log.New(i.LogFile, "", log.Ldate|log.Ltime)
	i.Logger.Printf("InfraredManager started.\n")
	return nil
}

func (i *InfraredManager) Close() {
	i.Logger.Printf("InfraredManager closed.\n")
	i.LogFile.Close()
}

func (i *InfraredManager) On() {
	i.Logger.Printf("starting command\n")
	done := false
	for !done {
		cmd := exec.Command("/bin/sh", "-c", "sudo /home/pi/go/src/joaowiciuk/juggernaut/c/./iron")
		if err := cmd.Start(); err != nil {
			i.Logger.Printf("starting command: %v\n", err)
			continue
		}
		if err := cmd.Wait(); err != nil {
			i.Logger.Printf("finishing command: %v\n", err)
			continue
		}
		done = true
	}
}

func (i *InfraredManager) Off() {
	done := false
	for !done {
		cmd := exec.Command("/bin/sh", "-c", "sudo /home/pi/go/src/joaowiciuk/juggernaut/c/./iroff")
		if err := cmd.Start(); err != nil {
			i.Logger.Printf("starting command: %v\n", err)
			continue
		}
		if err := cmd.Wait(); err != nil {
			i.Logger.Printf("finishing command: %v\n", err)
			continue
		}
		done = true
	}
}

func (i *InfraredManager) Receive() uint32 {
	i.Logger.Printf("receiving ir\n")
	done := false
	var received uint32
	for !done {
		cmd := exec.Command("/bin/sh", "-c", "sudo /home/pi/go/src/joaowiciuk/juggernaut/c/./irreceive")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			i.Logger.Printf("receiving ir: %s\n", err)
			continue
		}
		if err := cmd.Start(); err != nil {
			i.Logger.Printf("receiving ir: %s\n", err)
			continue
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(stdout)
		err = nil
		var unit string
		for err == nil {
			unit, err = buf.ReadString(0x0A)
			i.Logger.Printf("received ir unit: %s\n", unit)
		}
		if err := cmd.Wait(); err != nil {
			i.Logger.Printf("receiving ir: %s\n", err)
			continue
		}
		done = true
	}
	return received
}

func (i *InfraredManager) OperationHandler(w http.ResponseWriter, r *http.Request) {
	command := mux.Vars(r)["command"]
	switch command {
	case "on":
		i.On()
	default:
		i.Off()
	}
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
}

func (i *InfraredManager) ReceiveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	i.Logger.Println("Logger not working!")
	fmt.Fprintf(w, "%d", i.Receive())
}
