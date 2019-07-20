package main

import "time"

type Device struct {
	UUID        string
	Identifier  string
	Temperature float64
	LastUpdate  time.Time
}
