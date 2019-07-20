package main

import "time"

type Device struct {
	UUID        string    `json:"uuid"`
	Identifier  string    `json:"identifier"`
	Temperature float64   `json:"temperature"`
	LastUpdate  time.Time `json:"last_update"`
}
