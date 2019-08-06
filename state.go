package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

const (
	EquipmentPowerOn  = "PowerOn"
	EquipmentPowerOff = "PowerOff"
)

type StateManager struct {
	DatabaseManager *DatabaseManager
	LogFile         *os.File
	Logger          *log.Logger
	Websocket       *websocket.Conn
}

type EquipmentState struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

func NewStateManager() StateManager {
	return StateManager{}
}

func (s *StateManager) Initialize(logPath string, database *DatabaseManager) (err error) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	s.LogFile = f
	s.Logger = log.New(s.LogFile, "", log.Ldate|log.Ltime)
	s.DatabaseManager = database
	s.Logger.Printf("StateManager started.\n")
	return nil
}

func (s *StateManager) Close() {
	s.Logger.Printf("StateManager closed.\n")
	s.LogFile.Close()
	s.Websocket.Close()
}

func (s *StateManager) StateOf(equipment Equipment) (state EquipmentState) {
	state = EquipmentState{
		Name: equipment.Name,
	}
	switch equipment.Type {
	case EquipmentLamp:
		//TODO: Effectively read the lamp state
		state.State = EquipmentPowerOff
	default:
		//TODO: Implement for other equipments too
		state.State = EquipmentPowerOff
	}
	return
}

func (s *StateManager) Equipments() (states []EquipmentState) {
	//Initialize the list of states
	states = make([]EquipmentState, 0)

	//Get monitored equipments from DB
	equipments := s.DatabaseManager.ReadMonitoredEquipments()

	//For each equipment, read it's state
	for _, equipment := range equipments {
		state := s.StateOf(equipment)

		//Add the EquipmentState to the list of states
		states = append(states, state)
	}

	//Return the list of states
	return
}

func (s *StateManager) StateHandler(w http.ResponseWriter, r *http.Request) {
	s.Logger.Printf("state handler: request received.\n")
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.Logger.Printf("state handler: %v", err)
		return
	}
	defer conn.Close()
	for {
		states := s.Equipments()
		err := conn.WriteJSON(states)
		s.Logger.Printf("state handler: states sent: %v.\n", states)
		if err != nil {
			s.Logger.Printf("state handler: %v\n", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
	}
}
