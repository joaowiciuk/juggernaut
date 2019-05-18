package main

type adaptador interface {
	inicializar(endereco string) error
	finalizar()
	processar(...interface{}) (r *requisicao)
}
