package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

var logFile *os.File

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

	//Inicialização de adaptadores
	adaptadorBluetooth := newAdaptadorBluetooth()
	if err := adaptadorBluetooth.inicializar("registro_adaptador_bluetooth"); err != nil {
		log.Fatalf("Falha ao inicializar adaptador bluetooth\n")
	}
	defer adaptadorBluetooth.finalizar()

	adaptadorWifi := newAdaptadorWifi()
	if err := adaptadorWifi.inicializar("registro_adaptador_Wifi"); err != nil {
		log.Fatalf("Falha ao inicializar adaptador wifi\n")
	}
	defer adaptadorWifi.finalizar()

	auditorSimples := newAuditorSimples()
	if err := auditorSimples.inicializar("registro_auditor"); err != nil {
		log.Fatalf("Falha ao inicializar auditor\n")
	}
	defer auditorSimples.finalizar()

	//Manipuladores do adaptador Wifi
	adaptadorWifi.adicionarManipulador(olaHandler, "/ola", "GET")

	http.ListenAndServe(":8181", adaptadorWifi.roteador)
	log.Printf("Finalizando função principal...\n")
}

func olaHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Olá, mundo!\n")
}
