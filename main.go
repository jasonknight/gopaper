package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	gc "github.com/rthornton128/goncurses"
)

const (
	STARTUP = iota
)

func oops(s string) error {
	return errors.New(s)
}

var (
	sarg            = flag.String(`s`, `default value`, `document the option here`)
	logFilePath     = flag.String(`l`, `gopaper.log`, `the path to your chosen logfile`)
	yamlAdapterPath = flag.String(`a`, `../gopaper.db.yml`, `the adapter YAML for gopress`)
)
var Info *log.Logger
var Error *log.Logger
var mysql *MysqlAdapter

func init() {
	flag.Parse()
}
func main() {
	file, err := os.OpenFile(*logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Failed to open log file:", err)
		return
	}
	defer file.Close()
	Info = log.New(file, `[gopaper INF]:`, log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(file, `[gopaper ERR]:`, log.Ldate|log.Ltime|log.Lshortfile)
	Info.Println("Beginning")
	mysql, err = NewMysqlAdapterEx(*yamlAdapterPath)
	if err != nil {
		Error.Println(err)
		return
	}
	mysql.SetLogs(file)
	Info.Println("Database opened for reading")
	if err != nil {
		Error.Println(err)
		return
	}
	return
}



