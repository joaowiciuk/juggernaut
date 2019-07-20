package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/boltdb/bolt"
)

var logFile *os.File

const (
	UUID       = "56a01ff8-ce43-4b6f-9ad7-fa819a713fcf"
	Identifier = "SHC 0"
)

func init() {
	logFile, err := os.OpenFile("registro_principal", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	log.SetOutput(logFile)
	log.Printf("Juggernaut iniciado\n")
}

func main() {
	defer logFile.Close()
	log.Printf("Iniciando função principal\n")

	//Inicialização do banco de dados
	banco := newBanco()
	if err := banco.inicializar("registro_banco", "banco_de_dados.db", 0600, &bolt.Options{Timeout: 1 * time.Second}); err != nil {
		log.Fatalf("Falha ao inicializar auditor\n")
	}
	defer banco.finalizar()

	//Inicialização telemetria
	telemetria := NewTelemetria(banco)
	go telemetria.Comunicar()
	defer telemetria.Desligar()

	//Inicialização de adaptadores
	adaptadorBluetooth := newAdaptadorBluetooth()
	if err := adaptadorBluetooth.inicializar("registro_adaptador_bluetooth", banco); err != nil {
		log.Fatalf("Falha ao inicializar adaptador bluetooth\n")
	}
	defer adaptadorBluetooth.finalizar()

	adaptadorWifi := newAdaptadorWifi()
	if err := adaptadorWifi.inicializar("registro_adaptador_wifi", banco); err != nil {
		log.Fatalf("Falha ao inicializar adaptador wifi\n")
	}
	defer adaptadorWifi.finalizar()

	auditorSimples := newAuditorSimples()
	if err := auditorSimples.inicializar("registro_auditor"); err != nil {
		log.Fatalf("Falha ao inicializar auditor\n")
	}
	defer auditorSimples.finalizar()

	http.ListenAndServe(":8181", adaptadorWifi.roteador)
	log.Printf("Finalizando função principal...\n")
}
