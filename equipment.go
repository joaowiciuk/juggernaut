package main

const (
	EquipmentMotor = "motor"
	EquipmentLamp  = "lamp"
	EquipmentRoom  = "room"
)

type Equipment struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	StatePin int    `json:"state_pin"`
	RelayPin int    `json:"relay_pin"`
}
