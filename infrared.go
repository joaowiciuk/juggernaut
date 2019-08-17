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

func (i *InfraredManager) Send(pin, signal string) {
	done := false
	for !done {
		cli := fmt.Sprintf("sudo /home/pi/go/src/joaowiciuk/juggernaut/c/./irsend %s %s", pin, signal)
		cmd := exec.Command("/bin/sh", "-c", cli)
		if err := cmd.Start(); err != nil {
			i.Logger.Printf("sending ir signal: %v\n", err)
			continue
		}
		if err := cmd.Wait(); err != nil {
			i.Logger.Printf("sending ir signal: %v\n", err)
			continue
		}
		done = true
	}
}

func (i *InfraredManager) Receive() (received string) {
	done := false
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
		if err := cmd.Wait(); err != nil {
			i.Logger.Printf("receiving ir: %s\n", err)
			continue
		}
		err = nil
		var unit string
		for err == nil {
			unit, err = buf.ReadString(0x0A)
			i.Logger.Printf("received ir unit: %s\n", unit)
		}
		done = true
	}
	return
}

func (i *InfraredManager) SendHandler(w http.ResponseWriter, r *http.Request) {
	pin := mux.Vars(r)["pin"]
	signal := mux.Vars(r)["signal"]
	i.Send(pin, signal)
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
}

func (i *InfraredManager) ReceiveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	fmt.Fprintf(w, "%s", i.Receive())
}
