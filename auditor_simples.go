package main

import (
	"log"
	"os"
)

type auditorSimples struct {
	registro    *os.File
	registrador *log.Logger
}

func newAuditorSimples() (a *auditorSimples) {
	return &auditorSimples{}
}

func (as *auditorSimples) auditar(r requisicao) (ok bool) {
	if r.cliente == "" || r.token == "" {
		return false
	}
	r.p.acionar()
	return true
}

func (as *auditorSimples) inicializar(endereco string) (err error) {
	as.registro, err = os.OpenFile(endereco, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	as.registrador = log.New(as.registro, "", log.Ldate|log.Ltime)
	as.registrador.Printf("Auditor inicializado\n")
	return err
}

func (as *auditorSimples) finalizar() {
	as.registrador.Printf("Finalizando auditor...\n")
	as.registro.Close()
}
