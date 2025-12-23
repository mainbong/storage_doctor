package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// LogLevel represents the log level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var levelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
}

// Logger is the main logger struct
type Logger struct {
	level      LogLevel
	fileLogger *log.Logger
	consoleLogger *log.Logger
	file       *os.File
	logDir     string
}

var defaultLogger *Logger

// Init initializes the default logger
func Init(logDir string, level LogLevel) error {
	logger, err := NewLogger(logDir, level)
	if err != nil {
		return err
	}
	defaultLogger = logger
	return nil
}

// NewLogger creates a new logger instance
func NewLogger(logDir string, level LogLevel) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFile := filepath.Join(logDir, fmt.Sprintf("storage-doctor_%s.log", timestamp))

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	fileLogger := log.New(file, "", log.LstdFlags)
	consoleLogger := log.New(os.Stdout, "", log.LstdFlags)

	// Also create a symlink to latest.log for easy access
	latestLog := filepath.Join(logDir, "latest.log")
	os.Remove(latestLog) // Remove old symlink if exists
	os.Symlink(filepath.Base(logFile), latestLog)

	return &Logger{
		level:         level,
		fileLogger:    fileLogger,
		consoleLogger: consoleLogger,
		file:          file,
		logDir:        logDir,
	}, nil
}

// log writes a log message
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] %s: %s", timestamp, levelName, message)

	l.fileLogger.Println(logMessage)
	if level >= WARN {
		l.consoleLogger.Println(logMessage)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

// GetLogDir returns the log directory
func (l *Logger) GetLogDir() string {
	return l.logDir
}

// Package-level functions for default logger

// Debug logs a debug message using the default logger
func Debug(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debug(format, args...)
	}
}

// Info logs an info message using the default logger
func Info(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Info(format, args...)
	}
}

// Warn logs a warning message using the default logger
func Warn(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warn(format, args...)
	}
}

// Error logs an error message using the default logger
func Error(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Error(format, args...)
	}
}

// Close closes the default logger
func Close() error {
	if defaultLogger != nil {
		return defaultLogger.Close()
	}
	return nil
}
