package logger

import (
	"delivery/internal/types"
	"log"
	"os"
)

// MyLogger implements AppLogger
type MyLogger struct {
	Inf *log.Logger
	Ent *log.Logger
	Err *log.Logger
}

func New() *MyLogger {
	return &MyLogger{
		Inf: log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime),
		// likely want different output locations for Info and Delivery logs
		Ent: log.New(os.Stdout, "DELIVERY\t", log.Ldate|log.Ltime),
		Err: log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (l *MyLogger) Info(msg ...string) {
	l.Inf.Println(msg)
}

func (l *MyLogger) Error(msg ...string) {
	l.Err.Println(msg)
}

func (l *MyLogger) Entry(e *types.LogEntry) {
	l.Ent.Println(e.ToLog())
}
