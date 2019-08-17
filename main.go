package main

import (
	"log"
	"net/http"
	"os"
)

var logFile *os.File

func init() {
	logFile, err := os.OpenFile("log/main", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("opening log file: %v", err)
	}
	log.SetOutput(logFile)
	log.Printf("SHC started.\n")
}

func main() {
	log.Printf("main() started.\n")
	defer logFile.Close()

	//InfraredManager
	infraredManager := NewInfraredManager()
	if err := infraredManager.Initialize("log/infrared"); err != nil {
		log.Fatalf("main(): Initializing infraredManager: %v\n", err)
	}
	defer infraredManager.Close()

	//SecurityManager
	securityManager := NewSecurityManager()
	if err := securityManager.Initialize("log/security"); err != nil {
		log.Fatalf("main(): Initializing securityManager: %v\n", err)
	}
	defer securityManager.Close()

	//deviceManager
	deviceManager := NewDeviceManager()
	if err := deviceManager.Initialize("log/device"); err != nil {
		log.Fatalf("main(): Initializing deviceManager: %v\n", err)
	}
	defer deviceManager.Close()

	//Inicialização do banco de dados
	databaseManager := NewDatabaseManager()
	if err := databaseManager.Initialize("log/database", "DATABASE"); err != nil {
		log.Fatalf("main(): Initializing databaseManager: %v\n", err)
	}
	defer databaseManager.Close()

	//EquipmentManager
	equipmentManager := NewEquipmentManager()
	if err := equipmentManager.Initialize("log/equipment", databaseManager, deviceManager); err != nil {
		log.Fatalf("main(): Initializing equipmentManager: %v\n", err)
	}
	defer equipmentManager.Close()

	//wifiManager
	wifiManager := NewWifiManager()
	if err := wifiManager.Initialize("log/wifi", databaseManager); err != nil {
		log.Fatalf("main(): Initializing wifiManager: %v\n", err)
	}
	defer wifiManager.Close()
	wifiManager.AddHandler(equipmentManager.OperationHandler, "/api/equipment/{command}", "POST")
	wifiManager.AddHandler(equipmentManager.EquipmentHandler, "/api/equipment", "GET")
	wifiManager.AddHandler(infraredManager.SendHandler, "/api/infrared/send/{pin}/{signal}", "GET")
	wifiManager.AddHandler(infraredManager.ReceiveHandler, "/api/infrared/receive", "GET")

	//Inicialização telemetria
	telemetryManager := NewTelemetryManager()
	if err := telemetryManager.Initialize("log/telemetry", databaseManager, deviceManager); err != nil {
		log.Fatalf("main(): Initializing telemetryManager: %v\n", err)
	}
	defer telemetryManager.Close()
	go telemetryManager.Communicate()

	//bluetoothManager
	bluetoothManager := NewBluetoothManager()
	if err := bluetoothManager.Initialize("log/bluetooth", databaseManager, deviceManager, securityManager); err != nil {
		log.Fatalf("main(): Initializing bluetoothManager: %v\n", err)
	}
	defer bluetoothManager.Close()

	http.ListenAndServe(":8181", wifiManager.Router)
	log.Printf("main() finished.\n")
}
