package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	rpio "github.com/stianeikeland/go-rpio"
)

const (
	TypeMotor = "motor"
	TypeLamp  = "lamp"
	TypeRoom  = "room"
)

//	Responsibilities:
//	*	To handle equipment operation and state reading via Wifi - HTTP
//	EquipmentManager
type EquipmentManager struct {
	LogFile   *os.File
	Logger    *log.Logger
	Equipment []Equipment

	*DatabaseManager
}

func NewEquipmentManager() (e *EquipmentManager) {
	return &EquipmentManager{}
}

func (e *EquipmentManager) Initialize(logPath string, database *DatabaseManager) error {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	e.LogFile = f
	e.Logger = log.New(e.LogFile, "", log.Ldate|log.Ltime)
	e.DatabaseManager = database
	e.Logger.Printf("EquipmentManager started.\n")
	return nil
}

func (e *EquipmentManager) Close() {
	e.Logger.Printf("EquipmentManager closed.\n")
	e.LogFile.Close()
}

type Equipment struct {
	ID uint `gorm:"primary_key"`

	Name     string `json:"name"`
	Type     string `json:"type"`
	StatePin int    `json:"state_pin"`
	RelayPin int    `json:"relay_pin"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" sql:"index"`
}

const (
	EquipmentPowerOn  = "PowerOn"
	EquipmentPowerOff = "PowerOff"
)

type State struct {
	Name  string `json:"name"` //Equipment name
	Value string `json:"value"`
}

func (e *EquipmentManager) Refresh() {
	e.Equipment = e.DatabaseManager.ReadEquipment()
}

const (
	CommandToggle = "toggle"
	CommandOn     = "on"
	CommandOff    = "off"
)

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

	return
}

func (e *EquipmentManager) StateOf(equipment Equipment) (state State) {
	state = State{
		Name: equipment.Name,
	}
	switch equipment.Type {
	case TypeLamp:
		//TODO: Effectively read the lamp state
		state.Value = EquipmentPowerOff
	default:
		//TODO: Implement for other equipment too
		state.Value = EquipmentPowerOff
	}
	return
}

func (em *EquipmentManager) Equipments() (states []State) {
	//Initialize the list of states
	states = make([]State, 0)

	//Get monitored equipment from DB
	equipment := em.DatabaseManager.ReadEquipment()

	//For each equipment, read it's state
	for _, e := range equipment {
		state := em.StateOf(e)

		//Add the State to the list of states
		states = append(states, state)
	}

	//Return the list of states
	return
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
	w.WriteHeader(http.StatusOK)
}

func (e *EquipmentManager) StateHandler(w http.ResponseWriter, r *http.Request) {
	e.Logger.Printf("state handler: request received.\n")
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		e.Logger.Printf("state handler: %v", err)
		return
	}
	defer conn.Close()
	for {
		states := e.Equipments()
		err := conn.WriteJSON(states)
		e.Logger.Printf("state handler: states sent: %v.\n", states)
		if err != nil {
			e.Logger.Printf("state handler: %v\n", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
	}
}
