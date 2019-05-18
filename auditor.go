package main

type auditor interface {
	inicializar(endereco string) (err error)
	finalizar()
	auditar(r requisicao) (ok bool)
}
