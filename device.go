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
	*DatabaseManager
}

type Info struct {
	ID          int `json:"id" gorm:"primary_key"`
	DeviceID    int
	UUID        string `json:"uuid"`
	Identifier  string `json:"identifier"`
	Environment string `json:"environment"`
}

type Network struct {
	Inet  string `json:"inet"`
	IP    string `json:"ip"`
	Cloud string `json:"cloud"` //http://solutech.site
}

type Customer struct {
	ID       int `json:"id" gorm:"primary_key"`
	DeviceID int
	Name     string `json:"name"`
	Account  string `json:"account"`
	Password string `json:"password" gorm:"-"`
	Hash     string `json:"hash"`
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

func (d *DeviceManager) Initialize(logPath string, databaseManager *DatabaseManager) (err error) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	d.LogFile = f
	d.Logger = log.New(d.LogFile, "", log.Ldate|log.Ltime)
	d.DatabaseManager = databaseManager
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
		cmd := exec.Command("/bin/sh", "-c", "sudo iw dev wlan0 scan | awk -f /home/pi/Scripts/wifi.awk")

		//Saída padrão do comando
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			d.Logger.Printf("wifi: getting stdout: %v\n", err)
			time.Sleep(time.Second * 1)
			continue
		}

		//Inicia o comando porém não aguarda finalização
		if err := cmd.Start(); err != nil {
			d.Logger.Printf("wifi: starting command: %v\n", err)
			time.Sleep(time.Second * 1)
			continue
		}

		//Converte d saída do comando para string
		buf := new(bytes.Buffer)
		buf.ReadFrom(stdout)
		output := buf.String()
		d.Logger.Printf("Output: %s\n", output)

		//Aguarda até que o comando finalize
		if err := cmd.Wait(); err != nil {
			d.Logger.Printf("wifi: finishing command: %v\n", err)
			time.Sleep(time.Second * 1)
			continue
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

func (d *DeviceManager) AnalogVariance() float64 {
	done := false
	var analogVariance float64
	for !done {
		cmd := exec.Command("/bin/sh", "-c", "/home/pi/go/src/joaowiciuk/juggernaut/c/./avariance")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			d.Logger.Println(err)
			continue
		}
		if err := cmd.Start(); err != nil {
			d.Logger.Println(err)
			continue
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(stdout)
		output := buf.String()
		analogVariance, err = strconv.ParseFloat(output, 64)
		if err != nil {
			d.Logger.Println(err)
			continue
		}
		done = true
	}
	d.Logger.Printf("Analog variance: %.3f\n", analogVariance)
	return analogVariance
}

func (d *DeviceManager) Network() (network Network) {
	network = Network{
		Inet:  d.Inet(),
		IP:    d.IP(),
		Cloud: "http://solutech.site",
	}
	return
}

func (d *DeviceManager) Inet() (inet string) {
	done := false
	for !done {
		cmd := exec.Command("/bin/sh", "-c", "ifconfig | grep inet")
		stdout, _ := cmd.StdoutPipe()
		cmd.Start()
		buf := new(bytes.Buffer)
		buf.ReadFrom(stdout)
		output := buf.String()
		if err := cmd.Wait(); err != nil {
			d.Logger.Println(err)
			time.Sleep(time.Millisecond * 250)
			continue
		}
		re := regexp.MustCompile(`\ +inet (\d+\.\d+\.\d+\.\d+)\ +netmask\ +\d+\.\d+\.\d+\.\d+\ +broadcast\ +\d+\.\d+\.\d+\.\d+`)
		submatches := re.FindAllStringSubmatch(output, -1)
		inet = submatches[0][1]
		done = true
	}
	return
}

func (d *DeviceManager) IP() (ip string) {
	done := false
	for !done {
		cmd := exec.Command("/bin/sh", "-c", "curl ifconfig.co")
		stdout, _ := cmd.StdoutPipe()
		cmd.Start()
		buf := new(bytes.Buffer)
		buf.ReadFrom(stdout)
		output := buf.String()
		if err := cmd.Wait(); err != nil {
			d.Logger.Println(err)
			time.Sleep(time.Second * 3)
			continue
		}
		re := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)`)
		submatches := re.FindAllStringSubmatch(output, -1)
		ip = submatches[0][1]
		done = true
	}
	return
}
