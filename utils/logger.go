package utils

import (
	"log"
	"os"
	"time"
)

// Logger wraps the standard log package with structured logging
type Logger struct {
	logger *log.Logger
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
	}
}

// Info logs an info message
func (l *Logger) Info(v ...interface{}) {
	l.logger.SetPrefix("[INFO] ")
	l.logger.Println(append([]interface{}{time.Now().Format("2006-01-02 15:04:05")}, v...)...)
}

// Error logs an error message
func (l *Logger) Error(v ...interface{}) {
	l.logger.SetPrefix("[ERROR] ")
	l.logger.Println(append([]interface{}{time.Now().Format("2006-01-02 15:04:05")}, v...)...)
}

// Warn logs a warning message
func (l *Logger) Warn(v ...interface{}) {
	l.logger.SetPrefix("[WARN] ")
	l.logger.Println(append([]interface{}{time.Now().Format("2006-01-02 15:04:05")}, v...)...)
}

// Debug logs a debug message
func (l *Logger) Debug(v ...interface{}) {
	l.logger.SetPrefix("[DEBUG] ")
	l.logger.Println(append([]interface{}{time.Now().Format("2006-01-02 15:04:05")}, v...)...)
}