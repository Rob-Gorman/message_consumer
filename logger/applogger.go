package logger

import "delivery/types"

// Modularity for the logger is likely unnecessary, particularly because
// changes in logger implementation are unlikely, and if it happens,
// it will be for the sake of a different API. But it demonstrates the concept
type AppLogger interface {
	Info(...string)
	Error(...string)
	Entry(*types.LogEntry)
}
