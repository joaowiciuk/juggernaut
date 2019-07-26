package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

type adaptadorWifi struct {
	registro    *os.File
	registrador *log.Logger
	roteador    *mux.Router
	banco       *banco
}

func newAdaptadorWifi() (aw *adaptadorWifi) {
	return &adaptadorWifi{}
}

func (aw *adaptadorWifi) inicializar(endereco string, banco *banco) (err error) {
	f, err := os.OpenFile(endereco, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	aw.registro = f
	aw.registrador = log.New(aw.registro, "", log.Ldate|log.Ltime)
	aw.registrador.Printf("Inicializando adaptador wifi...\n")
	aw.roteador = mux.NewRouter()
	aw.banco = banco
	return nil
}

func (aw *adaptadorWifi) finalizar() {
	aw.registrador.Printf("Finalizando adaptador wifi...\n")
	aw.registro.Close()
}

func (aw *adaptadorWifi) adicionarRota(f http.HandlerFunc, endereço, método string) {
	aw.roteador.HandleFunc(endereço, f).Methods(método)
}
