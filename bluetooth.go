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
	"strconv"
	"time"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

//	Responsabilities:
//	*	To send and to receive anything related to the device and device configuration
//	BlueetoothManager
type BluetoothManager struct {
	LogFile *os.File
	Logger  *log.Logger
	Device  gatt.Device
	*DatabaseManager
	*DeviceManager
	*SecurityManager
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

func (bm *BluetoothManager) Initialize(logPath string, database *DatabaseManager,
	deviceManager *DeviceManager, security *SecurityManager) error {
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
	bm.DeviceManager = deviceManager
	bm.SecurityManager = security
	bm.Logger.Printf("BluetoothManager started.\n")
	return nil
}

func (bm *BluetoothManager) Close() {
	bm.Logger.Printf("BluetoothManager closed.\n")
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

	temperature := s.AddCharacteristic(gatt.MustParseUUID("aee5af4f-d1a8-4855-b770-b912519327d6"))
	temperature.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		done := false
		for !done {
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
			done = true
			time.Sleep(time.Second * 5)
		}
	})

	wifi := s.AddCharacteristic(gatt.MustParseUUID("351e784a-4099-405e-8031-e4b473e668a4"))
	wifi.HandleNotifyFunc(func(r gatt.Request, notifier gatt.Notifier) {
		for !notifier.Done() {
			wifi := bm.DeviceManager.Wifi()

			//Registra todos os wifi encontrados
			bm.Logger.Println("Wifi found:")
			for _, wifi := range wifi {
				bm.Logger.Println(wifi)
			}

			//Converte os Wifis para uma string JSON
			source, err := json.Marshal(wifi)
			if err != nil {
				bm.Logger.Printf("marshalling wifi: %v\n", err)
				break
			}
			reader := bytes.NewReader(source)

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
				bm.Logger.Printf("transf[:%d] = %q\n", k, transf[:k])

				//envia o buffer de transferência pelo notifier
				fmt.Fprintf(notifier, "%s", transf[:k])
			}

			//Intervalo para não estressar o dispositivo
			time.Sleep(1750 * time.Millisecond)
		}
	})

	device := s.AddCharacteristic(gatt.MustParseUUID("cb62a27b-c0fe-4003-a24b-4577ed4a697e"))
	device.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		var device Device
		if err := json.Unmarshal(data, &device); err != nil {
			bm.Logger.Printf("unmarshalling device %v\n", err)
			return gatt.StatusUnexpectedError
		}
		bm.DatabaseManager.UpdateDevice(device)
		return gatt.StatusSuccess
	})
	device.HandleNotifyFunc(func(r gatt.Request, notifier gatt.Notifier) {
		for !notifier.Done() {
			device := bm.DatabaseManager.ReadDevice()

			bm.Logger.Println("Device read:")
			bm.Logger.Println(device)

			source, err := json.Marshal(device)
			if err != nil {
				bm.Logger.Printf("marshalling device: %v\n", err)
				break
			}
			reader := bytes.NewReader(source)

			transf := make([]byte, 8)

			for {
				k, err := reader.Read(transf)
				if err == io.EOF {
					break
				}

				bm.Logger.Printf("transf[:%d] = %q\n", k, transf[:k])

				fmt.Fprintf(notifier, "%s", transf[:k])
			}

			time.Sleep(1750 * time.Millisecond)
		}
	})

	return s
}
