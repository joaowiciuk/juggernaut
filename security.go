package main

import (
	"log"
	"os"
)

//	Abstract responsibilities:
//	*	To refuse non authorized relay and infrared operations
//	*	To refuse non authorized device configuration
//	Concrete responsibilities:
//	*	To manipulate a role system
//	*	To generate and check tokens
//	SecurityManager
type SecurityManager struct {
	LogFile *os.File
	Logger  *log.Logger
}

func NewSecurityManager() *SecurityManager {
	return &SecurityManager{}
}

func (s *SecurityManager) Initialize(logPath string) (err error) {
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
