package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/boltdb/bolt"
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

	//Inicialização do banco de dados
	databaseManager := NewDatabaseManager()
	if err := databaseManager.Initialize("database_log", "database.db", 0600, &bolt.Options{Timeout: 1 * time.Second}); err != nil {
		log.Fatalf("main(): Initializing database: %v\n", err)
	}
	defer databaseManager.Close()

	//RelayManager
	relayManager := NewRelayManager()
	if err := relayManager.Initialize("relay_log"); err != nil {
		log.Fatalf("main(): Initializing relayManager: %v\n", err)
	}
	defer relayManager.Close()

	//bluetoothManager
	bluetoothManager := NewBluetoothManager()
	if err := bluetoothManager.Initialize("bluetooth_log", databaseManager); err != nil {
		log.Fatalf("main(): Initializing bluetoothManager: %v\n", err)
	}
	defer bluetoothManager.Close()

	//configurationManager
	configurationManager := NewConfigurationManager()
	if err := configurationManager.Initialize("configuration_log", databaseManager, bluetoothManager); err != nil {
		log.Fatalf("main(): Initializing configurationManager: %v\n", err)
	}
	defer configurationManager.Close()

	//wifiManager
	wifiManager := NewWifiManager()
	if err := wifiManager.Initialize("wifi_log", databaseManager); err != nil {
		log.Fatalf("main(): Initializing wifiManager: %v\n", err)
	}
	defer wifiManager.Close()
	wifiManager.AddHandler(relayManager.RelayHandler, "/api/relay", "GET")
	wifiManager.AddHandler(relayManager.NoWebSocketRelayHandler, "/relay/{pin}", "GET")
	wifiManager.AddHandler(relayManager.NoWebSocketInfraredHandler, "/infrared/{pin}", "GET")

	//Inicialização telemetria
	telemetryManager := NewTelemetryManager()
	if err := telemetryManager.Initialize("telemetry_log", databaseManager); err != nil {
		log.Fatalf("main(): Initializing telemetryManager: %v\n", err)
	}
	defer telemetryManager.Close()
	go telemetryManager.Communicate()

	http.ListenAndServe(":8181", wifiManager.Router)
	log.Printf("main() finished.\n")
}
