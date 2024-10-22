package xtreamcodes

import (
	"log"
	"os"
)

// var debugLoggingEnabled bool

// func init() {
// 	debugLoggingEnabled = os.Getenv("DEBUG_LOGGING") == "true"
// }

// Logs a message if debug logging is enabled
func debugLog(format string, v ...interface{}) {
	debugLoggingEnabled := os.Getenv("DEBUG_LOGGING") == "true"
	if debugLoggingEnabled {
		log.Printf("[DEBUG] "+format, v...)
	}
}
