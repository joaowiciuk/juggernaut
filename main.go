package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

var logger *log.Logger

func init() {
	logFile, err := os.OpenFile("log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer logFile.Close()
	logger = log.New(logFile, "", log.Lshortfile|log.LstdFlags)
	logger.Printf("Juggernaut iniciado\n")
}

func main() {
	logger.Printf("Iniciando função principal\n")
	router := mux.NewRouter()
	router.HandleFunc("/olá", olaHandler).Methods("GET")
	adaptadorBluetooth()
	http.ListenAndServe(":8181", router)
	logger.Printf("Finalizando função principal\n")
}

func olaHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Olá, mundo!\n")
}
