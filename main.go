package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func init() {
	logFile, err := os.OpenFile("log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.Printf("Juggernaut iniciado\n")
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/hello", helloWorldHandler).Methods("GET")
	//Comentário
	http.ListenAndServe(":8181", router)
}

func helloWorldHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, world!\n")
}
