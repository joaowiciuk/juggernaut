package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

type BluetoothManager struct {
	LogFile         *os.File
	Logger          *log.Logger
	Device          gatt.Device
	DatabaseManager *DatabaseManager
}

func NewBluetoothManager() (bm *BluetoothManager) {
	d, err := gatt.NewDevice(option.DefaultServerOptions...)
	if err != nil {
		log.Printf("creating bluetooth manager: %v\n", err)
	}
	return &BluetoothManager{
		Device: d,
	}
}

func (bm *BluetoothManager) Initialize(logPath string, database *DatabaseManager) error {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	bm.LogFile = f
	bm.Logger = log.New(bm.LogFile, "", log.Ldate|log.Ltime)
	if bm.Device == nil {
		bm.Logger.Printf("BlueetoothManager#Initialize(): nil Device")
	}
	bm.Device.Handle(
		gatt.CentralConnected(bm.OnConnect()),
		gatt.CentralDisconnected(bm.OnDisconnect()),
	)
	onStateChanged := func(d gatt.Device, s gatt.State) {
		bm.Logger.Printf("BlueetoothManager#Initialize(): State %s\n", s)
		switch s {
		case gatt.StatePoweredOn:

			service := bm.Service()
			d.AddService(service)
			d.AdvertiseNameAndServices("Solutech Home Connect", []gatt.UUID{service.UUID()})
		default:
		}
	}
	bm.Device.Init(onStateChanged)
	bm.DatabaseManager = database
	bm.Logger.Printf("BluetoothManager started.\n")
	return nil
}

func (bm *BluetoothManager) Finish() {
	bm.Logger.Printf("BluetoothManager finished.\n")
	bm.LogFile.Close()
}

func (bm *BluetoothManager) OnConnect() (f func(gatt.Central)) {
	return func(c gatt.Central) {
		bm.Logger.Printf("Device with ID %s connected.\n", c.ID())
	}
}

func (bm *BluetoothManager) OnDisconnect() (f func(gatt.Central)) {
	return func(c gatt.Central) {
		bm.Logger.Printf("Device with ID %s disconnected.\n", c.ID())
	}
}

func (bm *BluetoothManager) Service() *gatt.Service {
	s := gatt.NewService(gatt.UUID16(0x1815))

	ssidRequested := false

	readTemperature := s.AddCharacteristic(gatt.MustParseUUID("aee5af4f-d1a8-4855-b770-b912519327d6"))
	readTemperature.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		pending := true
		for pending {
			cmd := exec.Command("/bin/sh", "-c", "vcgencmd measure_temp")
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				bm.Logger.Println(err)
				break
			}
			if err := cmd.Start(); err != nil {
				bm.Logger.Println(err)
				break
			}
			buf := new(bytes.Buffer)
			buf.ReadFrom(stdout)
			output := buf.String()
			if err := cmd.Wait(); err != nil {
				bm.Logger.Println(err)
				break
			}
			re := regexp.MustCompile(`temp=(.*)'C`)
			submatches := re.FindAllStringSubmatch(output, -1)
			temp, err := strconv.ParseFloat(submatches[0][1], 64)
			if err != nil {
				bm.Logger.Println(err)
				break
			}
			fmt.Fprintf(rsp, "%f", temp)
			pending = false
		}
		time.Sleep(time.Second * 1)
	})

	requestSSID := s.AddCharacteristic(gatt.MustParseUUID("351e784a-4099-405e-8031-e4b473e668a4"))
	requestSSID.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		if len(data) == 1 && data[0] == 0x79 {
			ssidRequested = true
		} else {
			ssidRequested = false
		}
		return gatt.StatusSuccess
	})

	notifySSID := s.AddCharacteristic(gatt.MustParseUUID("34a97fc8-5118-4484-b022-0c8a467cd533"))
	notifySSID.HandleNotifyFunc(func(r gatt.Request, notifier gatt.Notifier) {
		for !notifier.Done() {
			for ssidRequested {
				//Comando para verificar redes wifi disponíveis
				cmd := exec.Command("/bin/sh", "-c", "sudo iw dev wlan0 scan | grep SSID")

				//Saída padrão do comando
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					bm.Logger.Println(err)
					break
				}

				//Inicia o comando porém não aguarda finalização
				if err := cmd.Start(); err != nil {
					bm.Logger.Println(err)
					break
				}

				//Converte bm saída do comando para string
				buf := new(bytes.Buffer)
				buf.ReadFrom(stdout)
				output := buf.String()

				//Aguarda até que o comando finalize
				if err := cmd.Wait(); err != nil {
					bm.Logger.Println(err)
					break
				}

				//Filtra bm saída do comando
				re := regexp.MustCompile(`\ *SSID:\ (.*)`)
				submatches := re.FindAllStringSubmatch(output, -1)
				type SSID struct {
					Lista []string `json:"ssids"`
				}
				ssid := SSID{
					Lista: make([]string, 0),
				}
				for _, submatch := range submatches {
					ssid.Lista = append(ssid.Lista, submatch[1])
				}

				//Nenhum SSID encontrado
				if len(ssid.Lista) == 0 {
					bm.Logger.Printf("BluetoothManager#Service(): No SSID found.\n")
					ssidRequested = false
					return
				}

				//Converte os SSIDs para uma string JSON codificada em base 64
				src, err := json.Marshal(ssid)
				if err != nil {
					bm.Logger.Printf("BluetoothManager#Service(): %v\n", err)
					ssidRequested = false
					return
				}
				size := ((4 * len(src) / 3) + 3) & ^3
				dst := make([]byte, size)
				base64.StdEncoding.Encode(dst, src)
				reader := bytes.NewReader(dst)

				//Registra todos os ssids encontrados
				for i, s := range ssid.Lista {
					bm.Logger.Printf("BluetoothManager#Service(): SSID[%d] = %s\n", i, s)
				}

				//Buffer de transferência para enviar em pedaços
				transf := make([]byte, 8)

				//Inicia bm transferência de ssidSource por mensagens do notifier
				// >> IMPORTANTE: para esta característica são permitidos apenas 8 bytes por mensagem <<
				for {
					k, err := reader.Read(transf)
					if err == io.EOF {
						ssidRequested = false
						break
					}

					//registra o buffer de transferência
					bm.Logger.Printf("transf[:%d] = %q\n", k, transf[:k])

					//envia o buffer de transferência pelo notifier
					fmt.Fprintf(notifier, "%s", transf[:k])
				}
			}
			//Intervalo para não estressar o dispositivo
			time.Sleep(time.Second * 1)
		}
	})

	readEnvironment := s.AddCharacteristic(gatt.MustParseUUID("ff39ae7e-61b6-4f67-af74-324e7af948bd"))
	readEnvironment.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		environment := bm.DatabaseManager.ReadEnvironment()
		rsp.SetStatus(gatt.StatusSuccess)
		fmt.Fprintf(rsp, "%s", environment)
	})

	writeEnvironment := s.AddCharacteristic(gatt.MustParseUUID("2f54b94a-a6fe-4d5f-a4ca-932a362eba10"))
	writeEnvironment.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		environment := string(data)
		bm.DatabaseManager.UpdateEnvironment(environment)
		return gatt.StatusSuccess
	})

	readIP := s.AddCharacteristic(gatt.MustParseUUID("02e9a221-8643-451e-ad92-deeec489c44b"))
	readIP.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		ip := bm.DatabaseManager.ReadIP()
		rsp.SetStatus(gatt.StatusSuccess)
		fmt.Fprintf(rsp, "%s", ip)
	})

	writeIP := s.AddCharacteristic(gatt.MustParseUUID("92e6b940-1ed5-43fb-b942-6ac51ad5d72d"))
	writeIP.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		ip := string(data)
		bm.DatabaseManager.UpdateIP(ip)
		return gatt.StatusSuccess
	})

	readIdentifier := s.AddCharacteristic(gatt.MustParseUUID("55cc9c0d-d42d-4f0f-850c-00b1809007e7"))
	readIdentifier.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		identifier := bm.DatabaseManager.ReadIdentifier()
		if identifier == "" {
			fmt.Fprint(rsp, "undefined")
		} else {
			fmt.Fprintf(rsp, "%s", identifier)
		}
	})

	writeIdentifier := s.AddCharacteristic(gatt.MustParseUUID("cde083d8-b20c-4709-b756-2f219a911994"))
	writeIdentifier.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		if bm.DatabaseManager.ReadIdentifier() == "" {
			identifier := string(data)
			bm.DatabaseManager.UpdateIdentifier(identifier)
		}
		return gatt.StatusSuccess
	})

	readUUID := s.AddCharacteristic(gatt.MustParseUUID("061e21d7-75bd-48fe-b0d5-b6237ef833c7"))
	readUUID.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		uuid := bm.DatabaseManager.ReadUUID()
		if uuid == "" {
			fmt.Fprint(rsp, "undefined")
		} else {
			fmt.Fprintf(rsp, "%s", uuid)
		}
	})

	writeUUID := s.AddCharacteristic(gatt.MustParseUUID("ecb2b207-78ab-44e1-a55e-dab0c6d4bf73"))
	writeUUID.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		if bm.DatabaseManager.ReadUUID() == "" {
			uuid := string(data)
			bm.DatabaseManager.UpdateUUID(uuid)
		}
		return gatt.StatusSuccess
	})

	return s
}
