package main

import (
	"log"
	"os"
)

type SecurityManager struct {
	LogFile *os.File
	Logger  *log.Logger
}

func NewSecurityManager() SecurityManager {
	return SecurityManager{}
}

func (s *SecurityManager) Initialize(logPath string, database *DatabaseManager, bluetooth *BluetoothManager) (err error) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	s.LogFile = f
	s.Logger = log.New(s.LogFile, "", log.Ldate|log.Ltime)
	s.Logger.Printf("SecurityManager started.\n")
	return nil
}

func (s *SecurityManager) Close() {
	s.Logger.Printf("SecurityManager closed.\n")
	s.LogFile.Close()
}
