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
	database := NewDatabase()
	if err := database.Initialize("database_log", "database.db", 0600, &bolt.Options{Timeout: 1 * time.Second}); err != nil {
		log.Fatalf("main(): Initializing database: %v\n", err)
	}
	defer database.Finish()

	//RelayManager
	relayManager := NewRelayManager()
	if err := relayManager.Initialize("relay_log"); err != nil {
		log.Fatalf("main(): Initializing relayManager: %v\n", err)
	}
	defer relayManager.Finish()

	//bluetoothManager
	/* bluetoothManager := NewBluetoothManager()
	if err := bluetoothManager.Initialize("bluetooth_log", database); err != nil {
		log.Fatalf("main(): Initializing bluetoothManager: %v\n", err)
	}
	defer bluetoothManager.Finish() */

	//wifiManager
	wifiManager := NewWifiManager()
	if err := wifiManager.Initialize("wifi_log", database); err != nil {
		log.Fatalf("main(): Initializing wifiManager: %v\n", err)
	}
	defer wifiManager.Finish()
	wifiManager.AddHandler(relayManager.RelayHandler, "/api/relay", "GET")

	//Inicialização telemetria
	telemetryManager := NewTelemetryManager()
	if err := telemetryManager.Initialize("telemetry_log", database); err != nil {
		log.Fatalf("main(): Initializing telemetryManager: %v\n", err)
	}
	defer telemetryManager.Finish()

	http.ListenAndServe(":8181", wifiManager.Router)
	log.Printf("main() finished.\n")
}
