package types

import (
	"fmt"
	"time"
)

type Message struct {
	Method    string
	URL       string
	Timestamp time.Time
	AckID     interface{}
}

type LogEntry struct {
	delivery     string
	response     string
	statusCode   int
	responseBody string
}

func NewLogEntry(d, r time.Time, sc int, rb string) *LogEntry {
	return &LogEntry{
		delivery:     d.Format(time.StampNano),
		response:     r.Format(time.StampNano),
		statusCode:   sc,
		responseBody: rb,
	}
}

func (e *LogEntry) ToLog() string {
	return fmt.Sprintf("%+v", *e)
}
