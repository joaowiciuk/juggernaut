package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"

	"github.com/gorilla/mux"
)

type adaptadorWifi struct {
	registro    *os.File
	registrador *log.Logger
	roteador    *mux.Router
	atualizador *websocket.Upgrader
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
	aw.adicionarManipulador(aw.manipuladorPrincipal(), "/", "GET")
	aw.atualizador = &websocket.Upgrader{}
	aw.banco = banco
	return nil
}

func (aw *adaptadorWifi) finalizar() {
	aw.registrador.Printf("Finalizando adaptador wifi...\n")
	aw.registro.Close()
}

func (aw *adaptadorWifi) processar(...interface{}) (req requisicao) {
	return
}

func (aw *adaptadorWifi) manipuladorPrincipal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		webSocket, err := aw.atualizador.Upgrade(w, r, nil)
		if err != nil {
			aw.registrador.Printf("Falha ao atualizar manipulador para websocket\n")
			return
		}
		defer webSocket.Close()
		//TODO: Como tratar o websocket a partir daqui?
	}
}

func (aw *adaptadorWifi) adicionarManipulador(f http.HandlerFunc, endereço, método string) {
	aw.roteador.HandleFunc(endereço, f).Methods(método)
}
