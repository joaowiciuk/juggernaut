package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

const (
	EnvironmentDevelopment = "dev"
	EnvironmentProduction  = "prod"
)

//	Responsibilities:
//	*	To collect device related information
//	DeviceManager
type DeviceManager struct {
	LogFile *os.File
	Logger  *log.Logger
}

type Device struct {
	ID uint `json:"id" gorm:"primary_key"`

	Info struct {
		UUID        string `json:"uuid"`
		Identifier  string `json:"identifier"`
		Environment string `json:"environment"`
	} `json:"device"`

	Network struct {
		Inet          string `json:"inet"`
		HostCloud     string `json:"host_cloud"`
		HostDebugging string `json:"host_debugging"`
	} `json:"network"`

	Customer struct {
		Name     string `json:"name"`
		Account  string `json:"account"`
		Password string `json:"password"`
	} `json:"customer"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" sql:"index"`
}

type Wifi struct {
	Name       string `json:"name"`
	Strength   string `json:"strength"`
	Encryption string `json:"encryption"`
}

func (s *Wifi) String() string {
	return fmt.Sprintf("Wifi{Name: %s, Strength: %s, Encryption: %s}", s.Name, s.Strength, s.Encryption)
}

func NewDeviceManager() *DeviceManager {
	return &DeviceManager{}
}

func (d *DeviceManager) Initialize(logPath string) (err error) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	d.LogFile = f
	d.Logger = log.New(d.LogFile, "", log.Ldate|log.Ltime)
	d.Logger.Printf("DeviceManager started.\n")
	return nil
}

func (d *DeviceManager) Close() {
	d.Logger.Printf("DeviceManager closed.\n")
	d.LogFile.Close()
}

func (d *DeviceManager) Wifis() (wifis []Wifi) {
	done := false
	wifis = make([]Wifi, 0)
	for !done {
		//Comando para verificar redes wifi disponíveis
		cmd := exec.Command("/bin/sh", "-d", "sudo iw dev wlan0 scan | awk -f /home/pi/Scripts/wifi.awk")

		//Saída padrão do comando
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			d.Logger.Println(err)
			break
		}

		//Inicia o comando porém não aguarda finalização
		if err := cmd.Start(); err != nil {
			d.Logger.Println(err)
			break
		}

		//Converte d saída do comando para string
		buf := new(bytes.Buffer)
		buf.ReadFrom(stdout)
		output := buf.String()
		d.Logger.Printf("Output: %s\n", output)

		//Aguarda até que o comando finalize
		if err := cmd.Wait(); err != nil {
			d.Logger.Println(err)
			break
		}

		//Filtra d saída do comando
		re := regexp.MustCompile(`\|@\| (.*) \|@\| (.*) \|@\| (.*) \|@\|`)
		submatches := re.FindAllStringSubmatch(output, -1)
		for _, submatch := range submatches {
			wifi := Wifi{
				Name:       submatch[1],
				Strength:   submatch[2],
				Encryption: submatch[3],
			}
			wifis = append(wifis, wifi)
		}

		done = true
		//Intervalo para não estressar o dispositivo
		time.Sleep(time.Second * 1)
	}
	return wifis
}

func (d *DeviceManager) Temperature() float64 {
	done := false
	var temp float64
	for !done {
		cmd := exec.Command("/bin/sh", "-c", "vcgencmd measure_temp")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			d.Logger.Println(err)
			time.Sleep(time.Second * 5)
			continue
		}
		if err := cmd.Start(); err != nil {
			d.Logger.Println(err)
			time.Sleep(time.Second * 5)
			continue
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(stdout)
		output := buf.String()
		if err := cmd.Wait(); err != nil {
			d.Logger.Println(err)
			time.Sleep(time.Second * 5)
			continue
		}
		re := regexp.MustCompile(`temp=(.*)'C`)
		submatches := re.FindAllStringSubmatch(output, -1)
		temp, err = strconv.ParseFloat(submatches[0][1], 64)
		if err != nil {
			d.Logger.Println(err)
			time.Sleep(time.Second * 5)
			continue
		}
		done = true
	}
	return temp
}
