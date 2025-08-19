package log

import (
	"fmt"
	"log"
)

// verbose controls the verbosity of the logging. Specifically
// it enables the Debug logging level.
var Verbose bool

// Debug logs a debug message against a specific scope. Debug
// logs will only be shown if verbose logging is enabled.
func Debug(scope string, msg string, args ...any) {
	if Verbose {
		msg = fmt.Sprintf(msg, args...)
		log.Printf("level='DEBUG' scope='%s' msg='%s'", scope, msg)
	}
}

// Info logs an informational message against a specific scope.
func Info(scope string, msg string, args ...any) {
	msg = fmt.Sprintf(msg, args...)
	log.Printf("level='INFO' scope='%s' msg='%s'", scope, msg)
}

// Warn logs a warning message against a specific scope.
func Warn(scope string, msg string, args ...any) {
	msg = fmt.Sprintf(msg, args...)
	log.Printf("level='WARN' scope='%s' msg='%s'", scope, msg)
}

// Error logs a non-blocking error message against a specific scope.
func Error(scope string, msg string, args ...any) {
	msg = fmt.Sprintf(msg, args...)
	log.Printf("level='ERROR' scope='%s' msg='%s'", scope, msg)
}

// Fatal logs a fatal error message against a specific scope and terminates the program.
func Fatal(scope string, msg string, args ...any) {
	msg = fmt.Sprintf(msg, args...)
	log.Fatalf("level='FATAL' scope='%s' msg='%s'", scope, msg)
}
