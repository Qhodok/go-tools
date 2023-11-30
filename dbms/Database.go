package dbms

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"sync"
)

const (
	POSTGRESQL = 1
	MYSQL      = 2
)

type Properties struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         string `json:"port"`
	MaxIdleConns int    `json:"max_idle_conns" mapstructure:"max_idle_conns"`
}

type DatabaseRepository struct {
	Database         *gorm.DB
	properties       *Properties
	connectionType   int
	connectionNumber int
	status           bool
	mutexStart       sync.Mutex
	mutexEnd         sync.Mutex
	mutexConn        sync.Mutex
	group            sync.WaitGroup
}

func New(properties *Properties, connectorType int) *DatabaseRepository {
	fmt.Println(properties)
	return &DatabaseRepository{properties: properties, connectionType: connectorType}

}

func (this *DatabaseRepository) Init(properties *Properties, connectorType int) {
	this.properties = properties
	this.connectionType = connectorType
}

func (this *DatabaseRepository) Connect() error {
	var err error
	if this.connectionType == POSTGRESQL {
		this.Database, err = gorm.Open("postgres", "host="+this.properties.Host+" port="+this.properties.Port+" user="+this.properties.Username+" dbname="+this.properties.Name+" password="+this.properties.Password+" sslmode=disable ")
	} else if this.connectionType == MYSQL {
		this.Database, err = gorm.Open("mysql", this.properties.Username+":"+this.properties.Password+"@tcp("+this.properties.Host+":"+this.properties.Port+")/"+this.properties.Name+"?charset=utf8&parseTime=True")
	} else {
		err = fmt.Errorf("dbms : unknown Database connector type %q", this.connectionType)
	}
	if err == nil {
		this.Database.DB().SetMaxIdleConns(this.properties.MaxIdleConns)
		this.status = true
		this.connectionNumber = 0
	} else {
		this.status = false
	}
	return err
}

func (this *DatabaseRepository) Begin() {
	this.mutexStart.Lock()
	if this.connectionNumber > this.properties.MaxIdleConns-10 {
		this.group.Add(1)
		this.group.Wait()
	}
	this.connectionNumber++
	this.mutexStart.Unlock()
}

func (this *DatabaseRepository) End() {
	this.mutexEnd.Lock()
	if this.connectionNumber > this.properties.MaxIdleConns-10 {
		this.group.Done()
	}
	this.connectionNumber--
	this.mutexEnd.Unlock()
}

func (this *DatabaseRepository) Populate(tableName string, result interface{}) (err error) {
	if this.status {
		this.Begin()
		err = this.Database.Table(tableName).Find(&result).Error
		this.End()
	} else {
		err = fmt.Errorf("dbms : not connected")
	}
	return
}

func (this *DatabaseRepository) Find(result interface{}, tableName string, colom string, value interface{}) (err error) {
	if this.status {
		this.Begin()
		err = this.Database.Table(tableName).Where(colom+" = ?", value).Find(&result).Error
		this.End()
	} else {
		err = fmt.Errorf("dbms : not connected")
	}
	return
}

func (this *DatabaseRepository) Select(result interface{}, tableName string, statement string, value ...interface{}) (err error) {
	if this.status {
		this.Begin()
		err = this.Database.Table(tableName).Where(statement, value...).Find(&result).Error
		this.End()
	} else {
		err = fmt.Errorf("dbms : not connected")
	}
	return
}
func (this *DatabaseRepository) SelectOrder(result interface{}, tableName string, statement string, order string, value ...interface{}) (err error) {
	if this.status {
		this.Begin()
		fmt.Println(result)
		err = this.Database.Table(tableName).Where(statement, value...).Find(&result).Order(order).Error
		this.End()
	} else {
		err = fmt.Errorf("dbms : not connected")
	}
	return
}

func (this *DatabaseRepository) Insert(table interface{}, tableName string) (err error) {
	if this.status {
		this.Begin()
		err = this.Database.Table(tableName).Create(&table).Error
		this.End()
	} else {
		err = fmt.Errorf("dbms : not connected")
	}
	return
}

func (this *DatabaseRepository) Update(tableName string, table interface{}, colom string, value interface{}) (err error) {
	if this.status {
		this.Begin()
		err = this.Database.Table(tableName).Where(colom+"=?", value).Update(&table).Error
		this.End()
	} else {
		err = fmt.Errorf("dbms : not connected")
	}
	return
}

func (this *DatabaseRepository) Delete(table interface{}) (err error) {
	if this.status {
		this.Begin()
		err = this.Database.Delete(table).Error
		this.End()
	} else {
		err = fmt.Errorf("dbms : not connected")
	}
	return
}
