package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

type adaptadorBluetooth struct {
	registro    *os.File
	registrador *log.Logger
	device      gatt.Device
}

func newAdaptadorBluetooth() (a *adaptadorBluetooth) {
	d, err := gatt.NewDevice(option.DefaultServerOptions...)
	if err != nil {
		log.Printf("Falha ao manipular dispositivo bluetooth, err: %s\n", err)
	}
	return &adaptadorBluetooth{
		device: d,
	}
}

func (a *adaptadorBluetooth) conexao() (f func(gatt.Central)) {
	return func(c gatt.Central) {
		a.registrador.Printf("Novo dispositivo conectado com ID %s\n", c.ID())
	}
}

func (a *adaptadorBluetooth) desconexao() (f func(gatt.Central)) {
	return func(c gatt.Central) {
		a.registrador.Printf("%s desconectou-se.\n", c.ID())
	}
}

func (a *adaptadorBluetooth) servicoPrincipal() *gatt.Service {
	s := gatt.NewService(gatt.MustParseUUID("19fc95c0-c111-11e3-9904-0002a5d5c51b"))
	s.AddCharacteristic(gatt.MustParseUUID("45fac9e0-c111-11e3-9246-0002a5d5c51b")).HandleWriteFunc(
		func(r gatt.Request, data []byte) (status byte) {
			a.processar(data)
			return gatt.StatusSuccess
		})
	return s
}

func (a *adaptadorBluetooth) servicoRedes() *gatt.Service {

	s := gatt.NewService(gatt.MustParseUUID("ac044f25-921b-4a9a-acaa-64c9fb77982a"))
	s.AddCharacteristic(gatt.MustParseUUID("4c3121dd-915b-4d54-a3e5-d8deb33114c3")).HandleReadFunc(
		func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
			cmd := exec.Command("/bin/sh", "-c", "sudo iw dev wlan0 scan | grep SSID")
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				fmt.Println(err)
				return
			}
			if err := cmd.Start(); err != nil {
				fmt.Println(err)
				return
			}
			buf := new(bytes.Buffer)
			buf.ReadFrom(stdout)
			output := buf.String()
			if err := cmd.Wait(); err != nil {
				fmt.Println(err)
				return
			}
			re := regexp.MustCompile(`SSID:\ (.*)`)
			ssids := re.FindAllString(output, -1)
			for _, ssid := range ssids {
				fmt.Println(ssid)
			}
			if len(ssids) < 2 {
				rsp.SetStatus(gatt.StatusUnexpectedError)
				rsp.Write([]byte("error: no ssid"))
				return
			}
			rsp.SetStatus(gatt.StatusSuccess)
			fmt.Fprintf(rsp, "%s", strings.Join(ssids, ","))
		})

	s.AddCharacteristic(gatt.MustParseUUID("87a040df-b13f-46d3-be03-ade57dcf1f07")).HandleNotifyFunc(
		func(r gatt.Request, n gatt.Notifier) {
			cmd := exec.Command("/bin/sh", "-c", "sudo iw dev wlan0 scan | grep SSID")
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				fmt.Println(err)
				return
			}
			if err := cmd.Start(); err != nil {
				fmt.Println(err)
				return
			}
			buf := new(bytes.Buffer)
			buf.ReadFrom(stdout)
			output := buf.String()
			if err := cmd.Wait(); err != nil {
				fmt.Println(err)
				return
			}
			re := regexp.MustCompile(`SSID:\ (.*)`)
			submatches := re.FindAllStringSubmatch(output, -1)
			ssids := make([]string, 0)
			for _, submatch := range submatches {
				ssids = append(ssids, submatch[0])
			}
			if len(ssids) < 2 {
				fmt.Printf("error: no ssid")
				return
			}
			for !n.Done() {
				for _, ssid := range ssids {
					fmt.Println(ssid)
					fmt.Fprintf(n, "%s", ssid)
					time.Sleep(time.Second)
				}
			}
		})

	s.AddCharacteristic(gatt.MustParseUUID("1c927b50-c116-11e3-8a33-0800200c9a66")).HandleNotifyFunc(
		func(r gatt.Request, n gatt.Notifier) {
			cnt := 0
			for !n.Done() {
				fmt.Fprintf(n, "Count: %d", cnt)
				cnt++
				time.Sleep(time.Second)
			}
		})

	return s
}

func (a *adaptadorBluetooth) inicializar(endereco string) error {
	f, err := os.OpenFile(endereco, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	a.registro = f
	a.registrador = log.New(a.registro, "", log.Ldate|log.Ltime)
	a.device.Handle(
		gatt.CentralConnected(a.conexao()),
		gatt.CentralDisconnected(a.desconexao()),
	)
	onStateChanged := func(d gatt.Device, s gatt.State) {
		a.registrador.Printf("Estado: %s\n", s)
		switch s {
		case gatt.StatePoweredOn:

			/* s1 := a.servicoPrincipal()
			d.AddService(s1) */

			s2 := a.servicoRedes()
			d.AddService(s2)

			d.AdvertiseNameAndServices("Solutech Home Connect 1", []gatt.UUID{ /* s1.UUID(), */ s2.UUID()})
			d.AdvertiseIBeacon(gatt.MustParseUUID("AA6062F098CA42118EC4193EB73CCEB6"), 1, 2, -59)
		default:
		}
	}
	a.registrador.Printf("Inicializando adaptador bluetooth...\n")
	a.device.Init(onStateChanged)
	return nil
}

func (a *adaptadorBluetooth) finalizar() {
	a.registrador.Printf("Finalizando adaptador bluetooth...\n")
	a.registro.Close()
}

func (a *adaptadorBluetooth) processar(dados []byte) (r *requisicao) {
	//TODO: especificar e implementar protocolo de comunicação por bluetooth
	s := string(dados)
	a.registrador.Printf("%d bytes recebidos\n", len(dados))
	a.registrador.Printf("Conteúdo: %s\n", s)
	return
}
