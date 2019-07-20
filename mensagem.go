package main

import "fmt"

type Mensagem struct {
	Contexto string                 `json:"contexto"`
	Conteudo map[string]interface{} `json:"conteudo"`
}

func (m Mensagem) String() string {
	return fmt.Sprintf("{Contexto: %s, Tamanho do conteúdo: %d valores}", m.Contexto, len(m.Conteudo))
}
