package main

import (
	"log"
	"net/http"
	"os"
)

var logFile *os.File

func init() {
	logFile, err := os.OpenFile("main_log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("opening log file: %v", err)
	}
	log.SetOutput(logFile)
	log.Printf("SHC started.\n")
}

func main() {
	log.Printf("main() started.\n")
	defer logFile.Close()

	//SecurityManager
	securityManager := NewSecurityManager()
	if err := securityManager.Initialize("log/security"); err != nil {
		log.Fatalf("main(): Initializing securityManager: %v\n", err)
	}
	defer securityManager.Close()

	//deviceManager
	deviceManager := NewDeviceManager()
	if err := deviceManager.Initialize("log/configuration"); err != nil {
		log.Fatalf("main(): Initializing deviceManager: %v\n", err)
	}
	defer deviceManager.Close()

	//Inicialização do banco de dados
	databaseManager := NewDatabaseManager()
	if err := databaseManager.Initialize("log/database", "cyttorak"); err != nil {
		log.Fatalf("main(): Initializing databaseManager: %v\n", err)
	}
	defer databaseManager.Close()

	//EquipmentManager
	equipmentManager := NewEquipmentManager()
	if err := equipmentManager.Initialize("log/equipment", databaseManager); err != nil {
		log.Fatalf("main(): Initializing equipmentManager: %v\n", err)
	}
	defer equipmentManager.Close()

	//wifiManager
	wifiManager := NewWifiManager()
	if err := wifiManager.Initialize("log/wifi", databaseManager); err != nil {
		log.Fatalf("main(): Initializing wifiManager: %v\n", err)
	}
	defer wifiManager.Close()
	wifiManager.AddHandler(equipmentManager.OperationHandler, "/equipment/{command}", "POST")

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
