package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"

	"github.com/paypal/gatt"
)

type ConfigurationManager struct {
	DatabaseManager  *DatabaseManager
	BluetoothManager *BluetoothManager
	LogFile          *os.File
	Logger           *log.Logger
}

type Configuration struct {
	UUID       string `json:"uuid"`
	Identifier string `json:"identifier"`
	Customer   struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	} `json:"customer"`
	Equipments []Equipment `json:"equipments"`
}

type SSID struct {
	Name       string `json:"name"`
	Strength   string `json:"strength"`
	Encryption string `json:"encryption"`
}

func (s *SSID) String() string {
	return fmt.Sprintf("SSID{Name: %s, Strength: %s, Encryption: %s}", s.Name, s.Strength, s.Encryption)
}

func NewConfigurationManager() ConfigurationManager {
	return ConfigurationManager{}
}

func (c *ConfigurationManager) Initialize(logPath string, database *DatabaseManager, bluetooth *BluetoothManager) (err error) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	c.LogFile = f
	c.Logger = log.New(c.LogFile, "", log.Ldate|log.Ltime)
	c.DatabaseManager = database
	c.BluetoothManager = bluetooth
	c.Logger.Printf("ConfigurationManager started.\n")
	return nil
}

func (c *ConfigurationManager) Close() {
	c.Logger.Printf("ConfigurationManager closed.\n")
	c.LogFile.Close()
}

func (c *ConfigurationManager) ListenConfiguration() {
	s := c.BluetoothManager.Service()
	characteristic := s.AddCharacteristic(gatt.MustParseUUID("88a00f38-6ee8-4e79-8302-855c9a6edac6"))
	characteristic.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		var configuration Configuration
		if err := json.Unmarshal(data, &configuration); err != nil {
			c.Logger.Printf("unmarshalling configuration: %v\n", err)
			return gatt.StatusUnexpectedError
		}
		c.SaveConfiguration(configuration)
		return gatt.StatusSuccess
	})
	c.BluetoothManager.Device.SetServices([]*gatt.Service{s})
}

func (c *ConfigurationManager) SaveConfiguration(configuration Configuration) {
	if err := c.DatabaseManager.UpdateConfiguration(configuration); err != nil {
		c.Logger.Printf("saving configuration: %v\n", err)
	}
	c.Logger.Printf("Configuration succesfuly saved!\n")
}

func (c *ConfigurationManager) SSIDS() (ssids []SSID) {
	done := false
	ssids = make([]SSID, 0)
	for !done {
		//Comando para verificar redes wifi disponíveis
		cmd := exec.Command("/bin/sh", "-c", "sudo iw dev wlan0 scan | grep SSID")

		//Saída padrão do comando
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			c.Logger.Println(err)
			break
		}

		//Inicia o comando porém não aguarda finalização
		if err := cmd.Start(); err != nil {
			c.Logger.Println(err)
			break
		}

		//Converte c saída do comando para string
		buf := new(bytes.Buffer)
		buf.ReadFrom(stdout)
		output := buf.String()

		//Aguarda até que o comando finalize
		if err := cmd.Wait(); err != nil {
			c.Logger.Println(err)
			break
		}

		//Filtra c saída do comando
		re := regexp.MustCompile(`\|@\| (.*) \|@\| (.*) \|@\| (.*) \|@\|`)
		submatches := re.FindAllStringSubmatch(output, -1)
		for _, submatch := range submatches {
			ssid := SSID{
				Name:       submatch[1],
				Strength:   submatch[2],
				Encryption: submatch[3],
			}
			ssids = append(ssids, ssid)
		}
	}
	return ssids
}

func (c *ConfigurationManager) SendSSID() {
	s := c.BluetoothManager.Service()
	characteristic := s.AddCharacteristic(gatt.MustParseUUID("4780e126-f320-4583-b2fd-dc9419e88aaf"))
	characteristic.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		var configuration Configuration
		if err := json.Unmarshal(data, &configuration); err != nil {
			c.Logger.Printf("unmarshalling configuration: %v\n", err)
			return gatt.StatusUnexpectedError
		}
		c.SaveConfiguration(configuration)
		return gatt.StatusSuccess
	})
	c.BluetoothManager.Device.SetServices([]*gatt.Service{s})
}
