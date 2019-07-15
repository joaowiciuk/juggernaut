package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/boltdb/bolt"
)

type banco struct {
	nucleo      *bolt.DB
	registro    *os.File
	registrador *log.Logger
}

func newBanco() *banco {
	return &banco{}
}

func (b *banco) inicializar(enderReg string, caminNucleo string, modArqNucleo os.FileMode, opcoesNucleo *bolt.Options) error {
	f, err := os.OpenFile(enderReg, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	b.registro = f
	b.registrador = log.New(b.registro, "", log.Ldate|log.Ltime)

	n, err := bolt.Open(caminNucleo, modArqNucleo, opcoesNucleo)
	if err != nil {
		return err
	}
	b.nucleo = n
	b.registrador.Printf("Inicializando banco...\n")
	return nil
}

func (b *banco) finalizar() {
	b.registrador.Printf("Encerrando banco de dados...\n")
	if err := b.nucleo.Close(); err != nil {
		b.registrador.Printf("Erro: %v\n", err)
	}
	b.registro.Close()
}

func (b *banco) salvarAmbiente(ambiente string) error {
	if ambiente != "DES" && ambiente != "PROD" {
		b.registrador.Printf("Erro: ambiente inválido fornecido: %s\n", ambiente)
		return errors.New("Ambiente inválido")
	}
	erroExterno := b.nucleo.Update(func(tx *bolt.Tx) error {
		balde, erroInterno := tx.CreateBucketIfNotExists([]byte("config"))
		if erroInterno != nil {
			b.registrador.Printf("Erro: não foi possível criar balde\n")
			return fmt.Errorf("criar balde: %s", erroInterno)
		}
		erroInterno = balde.Put([]byte("ambiente"), []byte(ambiente))
		return erroInterno
	})
	return erroExterno
}

func (b *banco) lerAmbiente() (ambiente string) {
	b.nucleo.View(func(tx *bolt.Tx) error {
		balde := tx.Bucket([]byte("config"))
		if balde == nil {
			ambiente = ""
		} else {
			ambiente = string(balde.Get([]byte("ambiente")))
		}
		return nil
	})
	return
}

func (b *banco) salvarIP(ip string) error {
	if ok, err := regexp.Match(`(?:[0-9]{1,3}\.){3}[0-9]{1,3}`, []byte(ip)); !ok || err != nil {
		b.registrador.Printf("Erro: IP inválido fornecido: %s\n", ip)
		return errors.New("IP inválido")
	}
	erroExterno := b.nucleo.Update(func(tx *bolt.Tx) error {
		balde, erroInterno := tx.CreateBucketIfNotExists([]byte("config"))
		if erroInterno != nil {
			b.registrador.Printf("Erro: não foi possível criar balde\n")
			return fmt.Errorf("criar balde: %s", erroInterno)
		}
		erroInterno = balde.Put([]byte("ip"), []byte(ip))
		return erroInterno
	})
	return erroExterno
}

func (b *banco) lerIP() (ip string) {
	b.nucleo.View(func(tx *bolt.Tx) error {
		balde := tx.Bucket([]byte("config"))
		if balde == nil {
			ip = ""
		} else {
			ip = string(balde.Get([]byte("ip")))
		}
		return nil
	})
	return
}

func (b *banco) salvarNomeCliente(nomeCliente string) error {
	if nomeCliente == "" {
		b.registrador.Printf("Erro: nome-cliente inválido fornecido: %s\n", nomeCliente)
		return errors.New("nome-cliente inválido")
	}
	erroExterno := b.nucleo.Update(func(tx *bolt.Tx) error {
		balde, erroInterno := tx.CreateBucketIfNotExists([]byte("config"))
		if erroInterno != nil {
			b.registrador.Printf("Erro: não foi possível criar balde\n")
			return fmt.Errorf("criar balde: %s", erroInterno)
		}
		erroInterno = balde.Put([]byte("nome-cliente"), []byte(nomeCliente))
		return erroInterno
	})
	return erroExterno
}

func (b *banco) lerNomeCliente() (nomeCliente string) {
	b.nucleo.View(func(tx *bolt.Tx) error {
		balde := tx.Bucket([]byte("config"))
		if balde == nil {
			nomeCliente = ""
		} else {
			nomeCliente = string(balde.Get([]byte("nome-cliente")))
		}
		return nil
	})
	return
}
