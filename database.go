package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"

	"github.com/boltdb/bolt"
)

const (
	ConfigurationBucket = "config"
	EquipmentBucket     = "equipment"

	EquipmentsKey          = "equipments"
	EnvironmentKey         = "environment"
	EnvironmentDevelopment = "dev"
	EnvironmentProduction  = "prod"

	IPKey = "ip"

	IdentifierKey = "identifier"

	UUIDKey = "uuid"
)

type DatabaseManager struct {
	kernel  *bolt.DB
	LogFile *os.File
	Logger  *log.Logger
}

func NewDatabaseManager() *DatabaseManager {
	return &DatabaseManager{}
}

func (dm *DatabaseManager) Initialize(logPath string, kernelPath string, kernelFileMode os.FileMode, kernelOptions *bolt.Options) error {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	dm.LogFile = f
	dm.Logger = log.New(dm.LogFile, "", log.Ldate|log.Ltime)

	n, err := bolt.Open(kernelPath, kernelFileMode, kernelOptions)
	if err != nil {
		return err
	}
	dm.kernel = n
	dm.Logger.Printf("DatabaseManager started.\n")
	return nil
}

func (dm *DatabaseManager) Close() {
	if err := dm.kernel.Close(); err != nil {
		dm.Logger.Printf("Erro: %v\n", err)
	}
	dm.Logger.Printf("DatabaseManager closed.\n")
	dm.LogFile.Close()
}

func (dm *DatabaseManager) UpdateEnvironment(environment string) error {
	if environment != EnvironmentDevelopment && environment != EnvironmentProduction {
		dm.Logger.Printf("DatabaseManager#UpdateEnvironment(): invalid environment %s.\n", environment)
		return errors.New("invalid environment")
	}
	externalError := dm.kernel.Update(func(tx *bolt.Tx) error {
		bucket, internalError := tx.CreateBucketIfNotExists([]byte(ConfigurationBucket))
		if internalError != nil {
			dm.Logger.Printf("DatabaseManager#UpdateEnvironment(): error creating bucket\n")
			return fmt.Errorf("creating bucket: %s", internalError)
		}
		internalError = bucket.Put([]byte(EnvironmentKey), []byte(environment))
		return internalError
	})
	return externalError
}

func (dm *DatabaseManager) ReadEnvironment() (environment string) {
	dm.kernel.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ConfigurationBucket))
		if bucket == nil {
			environment = ""
		} else {
			environment = string(bucket.Get([]byte(EnvironmentKey)))
		}
		return nil
	})
	return
}

func (dm *DatabaseManager) UpdateIP(ip string) error {
	if ok, err := regexp.Match(`(?:[0-9]{1,3}\.){3}[0-9]{1,3}`, []byte(ip)); !ok || err != nil {
		dm.Logger.Printf("DatabaseManager#UpdateIP(): invalid IP %s.\n", ip)
		return errors.New("invalid ip")
	}
	externalError := dm.kernel.Update(func(tx *bolt.Tx) error {
		bucket, internalError := tx.CreateBucketIfNotExists([]byte(ConfigurationBucket))
		if internalError != nil {
			dm.Logger.Printf("DatabaseManager#UpdateIP(): error creating bucket\n")
			return fmt.Errorf("creating bucket: %s", internalError)
		}
		internalError = bucket.Put([]byte(IPKey), []byte(ip))
		return internalError
	})
	return externalError
}

func (dm *DatabaseManager) ReadIP() (ip string) {
	dm.kernel.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ConfigurationBucket))
		if bucket == nil {
			ip = ""
		} else {
			ip = string(bucket.Get([]byte(IPKey)))
		}
		return nil
	})
	return
}

func (dm *DatabaseManager) UpdateIdentifier(identifier string) error {
	if identifier == "" {
		dm.Logger.Printf("DatabaseManager#UpdateIdentifier(): invalid identifier %s.\n", identifier)
		return errors.New("invalid identifier")
	}
	externalError := dm.kernel.Update(func(tx *bolt.Tx) error {
		bucket, internalError := tx.CreateBucketIfNotExists([]byte(ConfigurationBucket))
		if internalError != nil {
			dm.Logger.Printf("DatabaseManager#UpdateIdentifier(): error creating bucket\n")
			return fmt.Errorf("creating bucket: %s", internalError)
		}
		internalError = bucket.Put([]byte(IdentifierKey), []byte(identifier))
		return internalError
	})
	return externalError
}

func (dm *DatabaseManager) ReadIdentifier() (identifier string) {
	dm.kernel.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ConfigurationBucket))
		if bucket == nil {
			identifier = ""
		} else {
			identifier = string(bucket.Get([]byte(IdentifierKey)))
		}
		return nil
	})
	return
}

func (dm *DatabaseManager) UpdateUUID(uuid string) error {
	regex := `[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}`
	if ok, err := regexp.Match(regex, []byte(uuid)); !ok || err != nil {
		dm.Logger.Printf("DatabaseManager#UpdateUUID(): invalid UUID %s.\n", uuid)
		return errors.New("invalid uuid")
	}
	externalError := dm.kernel.Update(func(tx *bolt.Tx) error {
		bucket, internalError := tx.CreateBucketIfNotExists([]byte(ConfigurationBucket))
		if internalError != nil {
			dm.Logger.Printf("DatabaseManager#UpdateUUID(): error creating bucket\n")
			return fmt.Errorf("creating bucket: %s", internalError)
		}
		internalError = bucket.Put([]byte(UUIDKey), []byte(uuid))
		return internalError
	})
	return externalError
}

func (dm *DatabaseManager) ReadUUID() (uuid string) {
	dm.kernel.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ConfigurationBucket))
		if bucket == nil {
			uuid = ""
		} else {
			uuid = string(bucket.Get([]byte(UUIDKey)))
		}
		return nil
	})
	return
}

func (dm *DatabaseManager) AddMonitoredEquipment(equipment Equipment) error {
	if reflect.DeepEqual(equipment, Equipment{}) {
		dm.Logger.Printf("DatabaseManager#UpdateIdentifier(): invalid equipment %s.\n", equipment)
		return errors.New("invalid equipment")
	}
	externalError := dm.kernel.Update(func(tx *bolt.Tx) error {
		bucket, internalError := tx.CreateBucketIfNotExists([]byte(EquipmentBucket))
		if internalError != nil {
			dm.Logger.Printf("DatabaseManager#UpdateIdentifier(): error creating bucket\n")
			return fmt.Errorf("creating bucket: %s", internalError)
		}
		equipments := dm.ReadMonitoredEquipments()

		//Calculates new ID
		if len(equipments) == 0 {
			equipment.ID = 1
		} else {
			equipment.ID = 0
			for _, e := range equipments {
				if e.ID >= equipment.ID {
					equipment.ID = e.ID + 1
				}
			}
		}

		equipments = append(equipments, equipment)
		source, err := json.Marshal(equipments)
		if err != nil {
			dm.Logger.Printf("adding monitored equipment: %v\n", err)
		}
		internalError = bucket.Put([]byte(EquipmentsKey), source)
		return internalError
	})
	return externalError
}

func (dm *DatabaseManager) RemoveMonitoredEquipment(equipment Equipment) error {
	if reflect.DeepEqual(equipment, Equipment{}) {
		dm.Logger.Printf("DatabaseManager#UpdateIdentifier(): invalid equipment %s.\n", equipment)
		return errors.New("invalid equipment")
	}
	externalError := dm.kernel.Update(func(tx *bolt.Tx) error {
		bucket, internalError := tx.CreateBucketIfNotExists([]byte(EquipmentBucket))
		if internalError != nil {
			dm.Logger.Printf("DatabaseManager#UpdateIdentifier(): error creating bucket\n")
			return fmt.Errorf("creating bucket: %s", internalError)
		}
		equipments := dm.ReadMonitoredEquipments()

		var index int
		found := false
		for j, e := range equipments {
			if e.ID == equipment.ID {
				index = j
				found = true
			}
		}
		if found {
			equipments[index] = equipments[len(equipments)-1]
			equipments = equipments[:len(equipments)-1]
		}

		source, err := json.Marshal(equipments)
		if err != nil {
			dm.Logger.Printf("adding monitored equipment: %v\n", err)
		}
		internalError = bucket.Put([]byte(EquipmentsKey), source)
		return internalError
	})
	return externalError
}

func (dm *DatabaseManager) ReadMonitoredEquipments() (equipments []Equipment) {
	equipments = make([]Equipment, 0)
	source := make([]byte, 0)
	dm.kernel.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(EquipmentBucket))
		if bucket != nil {
			source = bucket.Get([]byte(EquipmentsKey))
		}
		return nil
	})
	if err := json.Unmarshal(source, equipments); err != nil {
		dm.Logger.Printf("getting monitored equipments: %v\n", err)
	}
	return
}

func (dm *DatabaseManager) UpdateConfiguration(configuration Configuration) error {
	return errors.New("TODO: Implement configuration persistence")
}
