package dao

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type Dao struct {
	DB *sql.DB
}

func New(c *Config) (*Dao, error) {
	d := &Dao{
		DB: NewMySQL(c),
	}

	err := d.Ping()
	if err != nil {
		return nil, err
	}

	return d, nil
}

// Ping check mysql health.
func (d *Dao) Ping() (err error) {
	return  d.DB.Ping()
}

// Close release all mysql resource .
func (d *Dao) Close() {
	d.DB.Close()
}

type Config struct {
	Addr		string
	Port		string
	UserName	string
	Password	string
	DB			string

	Active      int
	Idle        int
}

const DSN = "%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local"

func NewMySQL(c *Config) *sql.DB {

	dsn := fmt.Sprintf(DSN, c.UserName, c.Password, c.Addr, c.Port, c.DB)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(c.Active)
	db.SetMaxIdleConns(c.Idle)
	db.SetConnMaxLifetime(time.Hour)
	return db
}