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
	EquipmentOn   = "on"
	EquipmentOff  = "off"
	CommandToggle = "toggle"
	CommandOn     = "on"
	CommandOff    = "off"
)

//	Responsibilities:
//	*	To handle equipment operation and state reading via Wifi - HTTP
//	EquipmentManager
type EquipmentManager struct {
	LogFile *os.File
	Logger  *log.Logger

	*DatabaseManager
	*DeviceManager
}

func NewEquipmentManager() (e *EquipmentManager) {
	return &EquipmentManager{}
}

func (e *EquipmentManager) Initialize(logPath string, databaseManager *DatabaseManager, deviceManager *DeviceManager) error {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	e.LogFile = f
	e.Logger = log.New(e.LogFile, "", log.Ldate|log.Ltime)
	e.DatabaseManager = databaseManager
	e.DeviceManager = deviceManager
	e.Logger.Printf("EquipmentManager started.\n")
	return nil
}

func (e *EquipmentManager) Close() {
	e.Logger.Printf("EquipmentManager closed.\n")
	e.LogFile.Close()
}

type Equipment struct {
	ID uint `json:"id" gorm:"primary_key"`

	Name         string `json:"name"`
	Type         string `json:"type"`
	StateAddress int    `json:"state_address"`
	RelayPin     int    `json:"relay_pin"`
	State        string `json:"state" gorm:"-"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" sql:"index"`
}

func (e *EquipmentManager) Operate(equipment Equipment, command string) {
	if err := rpio.Open(); err != nil {
		e.Logger.Printf("opening rpio: %v\n", err)
		return
	}
	defer rpio.Close()
	pin := equipment.RelayPin
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

func (e *EquipmentManager) SetStateOf(equipment *Equipment) {
	switch equipment.Type {
	case TypeLamp:
		if equipment.ID == 1 && e.DeviceManager.AnalogVariance() > 0.005 {
			equipment.State = EquipmentOn
		}
		fallthrough
	default:
		//TODO: Implement for other equipment too
		equipment.State = EquipmentOff
	}
}

func (e *EquipmentManager) OperationHandler(w http.ResponseWriter, r *http.Request) {
	var equipment Equipment
	if err := json.NewDecoder(r.Body).Decode(&equipment); err != nil {
		e.Logger.Printf("decoding equipment: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	command := mux.Vars(r)["command"]
	e.Operate(equipment, command)
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
}

func (e *EquipmentManager) EquipmentHandler(w http.ResponseWriter, r *http.Request) {
	equipment := e.DatabaseManager.ReadEquipment()
	for i := range equipment {
		e.SetStateOf(&equipment[i])
	}
	w.Header().Add("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(equipment); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
