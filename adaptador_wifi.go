package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"

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
	aw.adicionarRota(aw.rotaPrincipal(), "/", "GET")
	aw.atualizador = &websocket.Upgrader{}
	aw.banco = banco
	ambiente := banco.lerAmbiente()
	var url string
	if ambiente == "DES" {
		url = "179.234.70.32:8081"
	} else if ambiente == "PROD" {
		url = "http://solutech.site/shc/telemetria"
	}
	ip := banco.lerIP()
	body := bytes.NewReader([]byte(ip))
	http.NewRequest("GET", url, body)
	return nil
}

func (aw *adaptadorWifi) finalizar() {
	aw.registrador.Printf("Finalizando adaptador wifi...\n")
	aw.registro.Close()
}

func (aw *adaptadorWifi) processar(...interface{}) (req requisicao) {
	return
}

func (aw *adaptadorWifi) rotaPrincipal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aw.atualizador.CheckOrigin = func(r *http.Request) bool { return true }
		webSocket, err := aw.atualizador.Upgrade(w, r, nil)
		if err != nil {
			aw.registrador.Printf("Falha ao atualizar rota para websocket\n")
			return
		}
		defer webSocket.Close()
		mensagens := make(chan Mensagem)
		go func() {
			for {
				tipoMensagem, p, err := webSocket.ReadMessage()
				if err != nil {
					aw.registrador.Printf("Falha ao ler mensagem: %v\n", err)
					mensagens <- Mensagem{}
					return
				}
				if tipoMensagem == websocket.BinaryMessage {
					reader := bytes.NewReader(p)
					var mensagem Mensagem
					json.NewDecoder(reader).Decode(&mensagem)
					aw.registrador.Printf("Mensagem recebida: %s\n", mensagem)
					mensagens <- mensagem
				}
			}
		}()
		go func() {
			for {
				aw.registrador.Printf("Iniciando leitura de temperatura...")
				cmd := exec.Command("/bin/sh", "-c", "vcgencmd measure_temp")
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					aw.registrador.Println(err)
				}
				if err := cmd.Start(); err != nil {
					aw.registrador.Println(err)
				}
				buf := new(bytes.Buffer)
				buf.ReadFrom(stdout)
				output := buf.String()
				if err := cmd.Wait(); err != nil {
					aw.registrador.Println(err)
					break
				}
				re := regexp.MustCompile(`temp=(.*)'C`)
				submatches := re.FindAllStringSubmatch(output, -1)
				value, err := strconv.ParseFloat(submatches[0][1], 64)
				if err != nil {
					aw.registrador.Println(err)
				}
				mensagem := Mensagem{
					Contexto: "telemetria",
					Conteudo: map[string]interface{}{"temperatura": value},
				}
				aw.registrador.Printf("Temperatura lida: %.2f\n", mensagem.Conteudo["temperatura"])
				json.NewEncoder(w).Encode(mensagem)
				aw.registrador.Printf("Mensagem enviada: %s\n", mensagem)
				if aw.banco.lerAmbiente() == "DES" {
					time.Sleep(3 * time.Second)
				} else if aw.banco.lerAmbiente() == "PROD" {
					time.Sleep(60 * time.Second)
				} else {
					time.Sleep(15 * time.Second)
				}
			}
		}()
	}
}

func (aw *adaptadorWifi) adicionarRota(f http.HandlerFunc, endereço, método string) {
	aw.roteador.HandleFunc(endereço, f).Methods(método)
}
