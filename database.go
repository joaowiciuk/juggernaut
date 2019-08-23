package main

import (
	"log"
	"os"
	"strings"

	"github.com/jinzhu/gorm"

	_ "github.com/jinzhu/gorm/dialects/postgres"
)

//	Responsibilities:
//	*	To persist and to recover reusable data structures
//	DatabaseManager
type DatabaseManager struct {
	Kernel  *gorm.DB
	LogFile *os.File
	Logger  *log.Logger
}

func NewDatabaseManager() *DatabaseManager {
	return &DatabaseManager{}
}

func (dm *DatabaseManager) Initialize(logPath string, env string) error {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	dm.LogFile = f
	dm.Logger = log.New(dm.LogFile, "", log.Ldate|log.Ltime)

	dm.Kernel = dm.Open(dm.URL(env))
	dm.Logger.Printf("DatabaseManager started.\n")
	return nil
}

func (dm *DatabaseManager) Close() {
	if err := dm.Kernel.Close(); err != nil {
		dm.Logger.Printf("Erro: %v\n", err)
	}
	dm.Logger.Printf("DatabaseManager closed.\n")
	dm.LogFile.Close()
}

func (dm *DatabaseManager) CreateEquipment(equipment Equipment) Equipment {
	dm.Kernel.Create(&equipment)
	return equipment
}

func (dm *DatabaseManager) ReadEquipment() []Equipment {
	var equipment []Equipment
	dm.Kernel.Find(&equipment)
	return equipment
}

func (dm *DatabaseManager) UpdateEquipment(equipment Equipment) Equipment {
	dm.Kernel.Save(&equipment)
	return equipment
}

func (dm *DatabaseManager) DeleteEquipment(equipment Equipment) Equipment {
	dm.Kernel.Delete(&equipment)
	return equipment
}

func (dm *DatabaseManager) ReadInfo() Info {
	var info Info
	dm.Kernel.First(&info)
	return info
}

func (dm *DatabaseManager) WriteInfo(info Info) Info {
	dm.Kernel.Save(&info)
	return info
}

func (dm *DatabaseManager) ReadCustomer() Customer {
	var customer Customer
	dm.Kernel.First(&customer)
	return customer
}

func (dm *DatabaseManager) WriteCustomer(customer Customer) Customer {
	dm.Kernel.Save(&customer)
	return customer
}

func (dm *DatabaseManager) Open(url string) *gorm.DB {
	db, err := gorm.Open("postgres", url)

	if err != nil {
		panic(err)
	}

	db.LogMode(true)

	db.AutoMigrate(&Equipment{})
	db.AutoMigrate(&Info{})
	db.AutoMigrate(&Customer{})

	return db
}

func (dm *DatabaseManager) URL(env string) string {
	databaseURL := os.Getenv(env)
	dm.Logger.Printf("database URL: %s\n", databaseURL)
	aux := strings.Split(databaseURL, "//")
	if len(aux) == 0 {
		panic("can not find database url")
	}
	aux = strings.Split(aux[1], "/")
	dbName := aux[1]
	aux = strings.Split(aux[0], ":")
	user := aux[0]
	port := aux[2]
	aux = strings.Split(aux[1], "@")
	password := aux[0]
	host := aux[1]
	replacer := strings.NewReplacer("{host}", host, "{port}", port, "{user}", user, "{dbname}", dbName, "{password}", password)
	return replacer.Replace("host={host} port={port} user={user} dbname={dbname} password={password}")
}
