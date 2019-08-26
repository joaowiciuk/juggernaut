package main

import (
	"log"
	"os"
)

//	Responsibilities:
//	*	To schedule future relay and infrared operations
//	ScheduleManager
type ScheduleManager struct {
	DatabaseManager *DatabaseManager
	LogFile         *os.File
	Logger          *log.Logger
}

const (
	ScheduleTypeRelay    = "TypeRelay"
	ScheduleTypeInfrared = "TypeInfrared"

	ScheduleFrequencySingle = "FrequencySingle"
	ScheduleFrequencyDayly  = "FrequencyDayly"
)

type Schedule struct {
	ID        int    `json:"id"`
	Pin       int    `json:"pin"`
	Type      string `json:"type"`
	Frequency string `json:"frequency"`
}

func NewScheduleManager() ScheduleManager {
	return ScheduleManager{}
}

func (s *ScheduleManager) Initialize(logPath string, database *DatabaseManager) (err error) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	s.LogFile = f
	s.Logger = log.New(s.LogFile, "", log.Ldate|log.Ltime)
	s.DatabaseManager = database
	s.Logger.Printf("ScheduleManager started.\n")
	return nil
}

func (s *ScheduleManager) Close() {
	s.Logger.Printf("ScheduleManager closed.\n")
	s.LogFile.Close()
}
