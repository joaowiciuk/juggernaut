package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

var logFile *os.File

func init() {
	logFile, err := os.OpenFile("log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	log.SetOutput(logFile)
	log.Printf("Juggernaut iniciado\n")
}

func main() {
	defer logFile.Close()
	log.Printf("Iniciando função principal\n")
	router := mux.NewRouter()
	router.HandleFunc("/olá", olaHandler).Methods("GET")
	adaptadorBluetooth()
	http.ListenAndServe(":8181", router)
	log.Printf("Finalizando função principal\n")
}

func olaHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Olá, mundo!\n")
}
