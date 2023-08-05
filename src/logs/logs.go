package logs

import (
	"fmt"
	"log"
	"os"
)

type Loggers struct {
	Info     *log.Logger
	Warn     *log.Logger
	Error    *log.Logger
	Critical *log.Logger
}

func New(logName string) *Loggers {
	lf, err := os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("Unable to open log file: %v", err))
	}

	l := Loggers{}
	l.Info = log.New(lf, "INFO: ", log.LstdFlags)
	l.Warn = log.New(lf, "WARN: ", log.LstdFlags)
	l.Error = log.New(lf, "ERROR: ", log.LstdFlags)
	l.Critical = log.New(lf, "CRIT: ", log.LstdFlags)
	return &l
}
