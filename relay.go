package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	rpio "github.com/stianeikeland/go-rpio"
)

const (
	TypeMotor     = "motor"
	TypeLamp      = "lamp"
	TypeRoom      = "room"
	RelayOn       = "on"
	RelayOff      = "off"
	CommandToggle = "toggle"
	CommandOn     = "on"
	CommandOff    = "off"
)
 
//	Responsibilities:
//	*	To handle relay operation and state reading via Wifi - HTTP
//	RelayManager
type RelayManager struct {
	LogFile *os.File
	Logger  *log.Logger

	*DatabaseManager
	*DeviceManager
}

func NewRelayManager() (e *RelayManager) {
	return &RelayManager{}
}

func (e *RelayManager) Initialize(logPath string, databaseManager *DatabaseManager, deviceManager *DeviceManager) error {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	e.LogFile = f
	e.Logger = log.New(e.LogFile, "", log.Ldate|log.Ltime)
	e.DatabaseManager = databaseManager
	e.DeviceManager = deviceManager
	e.Logger.Printf("RelayManager started.\n")
	return nil
}

func (e *RelayManager) Close() {
	e.Logger.Printf("RelayManager closed.\n")
	e.LogFile.Close()
}

type Relay struct {
	ID int `json:"id" gorm:"primary_key"`

	Name         string `json:"name"`
	Type         string `json:"type"`
	StateAddress int    `json:"state_address"`
	RelayPin     int    `json:"relay_pin" gorm:"unique"`
	State        string `json:"state" gorm:"-"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" sql:"index"`
}

func (e *RelayManager) Operate(relay Relay, command string) {
	if err := rpio.Open(); err != nil {
		e.Logger.Printf("opening rpio: %v\n", err)
		return
	}
	defer rpio.Close()
	pin := relay.RelayPin
	rpioPin := rpio.Pin(pin)
	rpioPin.Output()
	switch command {
	case CommandToggle:
		rpioPin.Toggle()
		e.Logger.Printf("Toggle relay on pin %d.\n", pin)
	case CommandOn:
		rpioPin.Low()
		e.Logger.Printf("Switch on relay on pin %d.\n", pin)
	case CommandOff:
		rpioPin.High()
		e.Logger.Printf("Switch off relay on pin %d.\n", pin)
	default:
		rpioPin.Toggle()
		e.Logger.Printf("Toggle relay on pin %d.\n", pin)
	}
}

func (e *RelayManager) SetStateOf(relay *Relay) {
	switch relay.Type {
	//TODO: Implement for other relay too
	case TypeLamp:
		analogVariance := e.DeviceManager.AnalogVariance()
		e.Logger.Printf("Analog variance: %.3f\n", analogVariance)
		if relay.ID == 1 && analogVariance > 0.006 {
			relay.State = RelayOn
		} else {
			relay.State = RelayOff
		}
	default:
		relay.State = RelayOff
	}
}

func (e *RelayManager) OperationHandler(w http.ResponseWriter, r *http.Request) {
	var relay Relay
	if err := json.NewDecoder(r.Body).Decode(&relay); err != nil {
		e.Logger.Printf("decoding relay: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	command := mux.Vars(r)["command"]
	e.Operate(relay, command)
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
}

func (e *RelayManager) RelayHandler(w http.ResponseWriter, r *http.Request) {
	relay := e.DatabaseManager.ReadRelay()
	for i := range relay {
		e.SetStateOf(&relay[i])
	}
	w.Header().Add("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(relay); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
