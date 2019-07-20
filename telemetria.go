package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type Telemetria struct {
	Banco       *banco
	Registro    *os.File
	Registrador *log.Logger
	Websocket   *websocket.Conn
}

func NewTelemetria(b *banco) Telemetria {
	var t Telemetria
	arquivo, err := os.OpenFile("registro_telemetria", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("Não foi possível inicializar telemetria: %v\n", err)
	}
	t.Registro = arquivo
	t.Registrador = log.New(arquivo, "", log.Ldate|log.Ltime)
	t.Banco = b
	return t
}

func (t *Telemetria) Desligar() {
	t.Registrador.Printf("Desligando telemetria...\n")
	t.Registro.Close()
	t.Websocket.Close()
}

func (t *Telemetria) Temperatura() float64 {
	cmd := exec.Command("/bin/sh", "-c", "vcgencmd measure_temp")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Registrador.Println(err)
		return 0
	}
	if err := cmd.Start(); err != nil {
		t.Registrador.Println(err)
		return 0
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(stdout)
	output := buf.String()
	if err := cmd.Wait(); err != nil {
		t.Registrador.Println(err)
		return 0
	}
	re := regexp.MustCompile(`temp=(.*)'C`)
	submatches := re.FindAllStringSubmatch(output, -1)
	value, err := strconv.ParseFloat(submatches[0][1], 64)
	if err != nil {
		t.Registrador.Println(err)
		return 0
	}
	return value
}

func (t *Telemetria) Comunicar() {
	t.Registrador.Printf("Telemetria Comunicar()\n")
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	ambiente := t.Banco.lerAmbiente()
	var host string
	if ambiente == "DES" {
		host = "179.234.70.32:8081"
	} else if ambiente == "PROD" {
		host = "http://solutech.site"
	}
	u := url.URL{Scheme: "ws", Host: host, Path: "/shc/telemetria"}
	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Registrador.Fatal("dial:", err)
		return
	}
	defer c.Close()
	t.Websocket = c
	t.Registrador.Printf("Telemetria inicializada")
	var ticker *time.Ticker
	if t.Banco.lerAmbiente() == "DES" {
		ticker = time.NewTicker(5 * time.Second)
	} else if t.Banco.lerAmbiente() == "PROD" {
		ticker = time.NewTicker(60 * time.Second)
	} else {
		ticker = time.NewTicker(15 * time.Second)
	}
	defer ticker.Stop()
	isUp := true
	for isUp {
		select {
		case instant := <-ticker.C:
			//Writing
			mensagem := Mensagem{
				Contexto: "telemetria",
				Conteudo: make(map[string]interface{}),
			}
			mensagem.Conteudo["temperatura"] = t.Temperatura()
			mensagem.Conteudo["tempo"] = instant
			dados, err := json.Marshal(mensagem)
			if err != nil {
				t.Registrador.Println("codificar mensagem:", err)
				isUp = false
			}
			err = c.WriteMessage(websocket.BinaryMessage, dados)
			if err != nil {
				log.Println("write:", err)
				isUp = false
			}

			//Reading
			_, message, err := c.ReadMessage()
			if err != nil {
				t.Registrador.Println("Ao receber mensagem: ", err)
				isUp = false
			}
			t.Registrador.Printf("Recebido: %s", message)
		case <-interrupt:
			log.Println("interrupt")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				isUp = false
			}
			select {
			case <-time.After(time.Second):
			}
			isUp = false
		}
	}
	t.Comunicar()
}
