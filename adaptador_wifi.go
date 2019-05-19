package main

import (
	"log"
	"os"
)

type adaptadorWifi struct {
	registro    *os.File
	registrador *log.Logger
}

func newAdaptadorWifi() (aw *adaptadorWifi) {
	return &adaptadorWifi{}
}

func (aw *adaptadorWifi) inicializar(endereco string) (err error) {
	f, err := os.OpenFile(endereco, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	aw.registro = f
	aw.registrador = log.New(aw.registro, "", log.Ldate|log.Ltime)
	aw.registrador.Printf("Inicializando adaptador wifi...\n")
	return nil
}

func (aw *adaptadorWifi) finalizar() {
	aw.registrador.Printf("Finalizando adaptador wifi...\n")
	aw.registro.Close()
}

func (aw *adaptadorWifi) processar(...interface{}) (req requisicao) {
	return
}
