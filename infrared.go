package main

import (
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
