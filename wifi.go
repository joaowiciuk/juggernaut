package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

type WifiManager struct {
	LogFile         *os.File
	Logger          *log.Logger
	Router          *mux.Router
	DatabaseManager *DatabaseManager
}

func NewWifiManager() (wm *WifiManager) {
	return &WifiManager{}
}

func (wm *WifiManager) Initialize(logPath string, database *DatabaseManager) (err error) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	wm.LogFile = f
	wm.Logger = log.New(wm.LogFile, "", log.Ldate|log.Ltime)
	wm.Router = mux.NewRouter()
	wm.DatabaseManager = database
	wm.Logger.Printf("WifiManager started.\n")
	return nil
}

func (wm *WifiManager) Close() {
	wm.Logger.Printf("WifiManager closed.\n")
	wm.LogFile.Close()
}

func (wm *WifiManager) AddHandler(f http.HandlerFunc, route, method string) {
	wm.Router.HandleFunc(route, f).Methods(method)
}
