package auth

import (
	"fmt"
	"github.com/go-xorm/xorm"
	"time"
	"xorm.io/core"
)

type RDBStore interface {
	Engine() *xorm.Engine
	CheckTableExistsAndCreate(bean interface{}) error
}

type mysqlGdbc struct {
	engine *xorm.Engine
}

func NewMysqlGdbc (dataSourceName dataSourceName) (*mysqlGdbc, error) {
	engine, err := xorm.NewEngine("mysql", string(dataSourceName))
	if err != nil {
		return nil, err
	}
	defaultTZ := "Asia/Seoul" // it may not be changed.
	engine.TZLocation, _ = time.LoadLocation(defaultTZ)
	engine.SetTableMapper(core.GonicMapper{})
	return &mysqlGdbc{engine: engine}, nil
}

func (mysql *mysqlGdbc) Engine() *xorm.Engine {
	return mysql.engine
}

func (mysql *mysqlGdbc) CheckTableExistsAndCreate(bean interface{}) error {
	if exists, err := mysql.Engine().IsTableExist(bean); err != nil {
		return err
	} else {
		if !exists {
			err = mysql.Engine().Charset("utf8mb4").CreateTable(bean)
			if err != nil {
				return err
			}
		}
	}
	return nil
}


type dataSourceName string

func NewMysqlDataSourceName (mysqlUser, password, protocol, host, port, db string) dataSourceName {
	if protocol == "" {
		protocol = "tcp"
	}
	if port == "" {
		port = "3306"
	}
	return dataSourceName(fmt.Sprintf("%s:%s@%s(%s:%s)/%s",mysqlUser,password,protocol,host,port,db))
}
