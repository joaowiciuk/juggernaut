package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"time"

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
		if err := c.DatabaseManager.UpdateConfiguration(configuration); err != nil {
			c.Logger.Printf("saving configuration: %v\n", err)
		}
		c.Logger.Printf("Configuration succesfuly saved!\n")
		return gatt.StatusSuccess
	})
	c.BluetoothManager.Device.SetServices([]*gatt.Service{s})
}

func (c *ConfigurationManager) SSIDS() (ssids []SSID) {
	done := false
	ssids = make([]SSID, 0)
	for !done {
		//Comando para verificar redes wifi disponíveis
		cmd := exec.Command("/bin/sh", "-c", "sudo iw dev wlan0 scan | awk -f /home/pi/Scripts/wifi.awk")

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
		c.Logger.Println("Output:")
		submatches := re.FindAllStringSubmatch(output, -1)
		for _, submatch := range submatches {
			ssid := SSID{
				Name:       submatch[1],
				Strength:   submatch[2],
				Encryption: submatch[3],
			}
			c.Logger.Println(ssid)
			ssids = append(ssids, ssid)
		}

		//Intervalo para não estressar o dispositivo
		time.Sleep(time.Second * 1)
	}
	c.Logger.Printf("SSIDs: %s\n", ssids)
	return ssids
}

func (c *ConfigurationManager) HandleSSIDSRequests() {
	s := c.BluetoothManager.Service()
	characteristic := s.AddCharacteristic(gatt.MustParseUUID("4780e126-f320-4583-b2fd-dc9419e88aaf"))
	characteristic.HandleNotifyFunc(func(r gatt.Request, notifier gatt.Notifier) {
		for !notifier.Done() {
			ssids := c.SSIDS()

			//Converte os SSIDs para uma string JSON codificada em base 64
			source, _ := json.Marshal(ssids)
			reader := bytes.NewReader(source)

			//Registra todos os ssids encontrados
			for _, ssid := range ssids {
				c.Logger.Println(ssid)
			}

			//Buffer de transferência para enviar em pedaços
			transf := make([]byte, 8)

			//Inicia transferência mensagens do notifier
			// >> IMPORTANTE: para esta característica são permitidos apenas 8 bytes por mensagem <<
			for {
				k, err := reader.Read(transf)
				if err == io.EOF {
					break
				}

				//registra o buffer de transferência
				c.Logger.Printf("transf[:%d] = %q\n", k, transf[:k])

				//envia o buffer de transferência pelo notifier
				fmt.Fprintf(notifier, "%s", transf[:k])
			}

			//Intervalo para não estressar o dispositivo
			time.Sleep(time.Second * 1)
		}
	})
	c.BluetoothManager.Device.SetServices([]*gatt.Service{s})
}
