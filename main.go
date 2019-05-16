package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

func init() {
	logFile, err := os.OpenFile("log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	data := time.Now()
	dataTexto := data.Format("Monday 02-01-2006 15:04:05")
	log.Printf("Juggernaut iniciado em %s\n", dataTexto)
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/hello", helloWorldHandler).Methods("GET")
	http.ListenAndServe(":8181", router)
}

func helloWorldHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, world!\n")
}
