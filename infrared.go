package main

/*
#cgo CFLAGS: -I${SRCDIR}/c -g -Wall
#cgo LDFLAGS: -L${SRCDIR}/c -linfrared
#include <stdlib.h>
#include <infrared.h>
*/
import "C"
import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"unsafe"

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
	cPin := C.CString(pin)
	defer C.free(unsafe.Pointer(cPin))

	cSignal := C.CString(signal)
	defer C.free(unsafe.Pointer(cSignal))

	undefined := C.send(cPin, cSignal)
	response := int(undefined)

	i.Logger.Printf("sending ir signal: received from c function: %d\n", response)

	/* done := false
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
	} */
}

func (i *InfraredManager) Receive() (received string) {
	cmd := exec.Command("/bin/sh", "-c", "sudo /home/pi/go/src/joaowiciuk/juggernaut/c/./irreceive")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		i.Logger.Printf("receiving ir: %s\n", err)
		return "0"
	}
	if err := cmd.Start(); err != nil {
		i.Logger.Printf("receiving ir: %s\n", err)
		return "0"
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(stdout)
	if err := cmd.Wait(); err != nil {
		i.Logger.Printf("receiving ir: %s\n", err)
		return "0"
	}
	var previousPolarity string
	var previousMicros int64
	tolerance := int64(250)
	for {
		symbol, err := buf.ReadString(0x0A)
		if err == io.EOF {
			break
		}

		if symbol[0:len(symbol)-1] == "end" {
			break
		}

		if symbol[0:len(symbol)-1] == "start" {
			continue
		}

		currentPolarity := symbol[0:1]
		currentMicros, _ := strconv.ParseInt(symbol[2:len(symbol)-1], 10, 64)

		if currentPolarity == "0" && (currentMicros < 562+tolerance && currentMicros > 562-tolerance) &&
			previousPolarity == "1" && (previousMicros < 562+tolerance && previousMicros > 562-tolerance) {
			received = received + "0"
		}
		if currentPolarity == "0" && (currentMicros < 1687+tolerance && currentMicros > 1687-tolerance) &&
			previousPolarity == "1" && (previousMicros < 562+tolerance && previousMicros > 562-tolerance) {
			received = received + "1"
		}

		i.Logger.Printf("%s %d\n", currentPolarity, currentMicros)

		previousPolarity = currentPolarity
		previousMicros = currentMicros
	}
	return received
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
