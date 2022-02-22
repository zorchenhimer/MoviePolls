package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type LogLevel string

const (
	LLSilent LogLevel = "silent" // absoultely nothing
	LLError  LogLevel = "error"  // only log errors
	LLInfo   LogLevel = "info"   // log info messages (not quite debug, but not chat)
	LLDebug  LogLevel = "debug"  // log everything
)

const (
	logPrefixError string = "[ERROR] "
	logPrefixChat  string = "[CHAT] "
	logPrefixInfo  string = "[INFO] "
	logPrefixDebug string = "[DEBUG] "
)

type Logger struct {
	lInfo  *log.Logger
	lError *log.Logger
	lDebug *log.Logger
}

func (l *Logger) Info(s string, v ...interface{}) {
	if l.lInfo == nil {
		return
	}

	l.lInfo.Printf(s+"\n", v...)
}

func (l *Logger) Error(s string, v ...interface{}) {
	if l.lError == nil {
		return
	}

	l.lError.Printf(s+"\n", v...)
}

func (l *Logger) Debug(s string, v ...interface{}) {
	if l.lDebug == nil {
		return
	}

	l.lDebug.Printf(s+"\n", v...)
}

func NewLogger(level LogLevel, file string) (*Logger, error) {
	l := &Logger{}
	baseDir := filepath.Dir(file)
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("Unable to create log directory %q: %w", baseDir, err)
	}

	switch LogLevel(strings.ToLower(string(level))) {
	case LLSilent:
		fmt.Println("[SILENT] Nothing to see here, please leave the area!")
		return l, nil

	case LLError:
		fmt.Println(logPrefixError + "Logging enabled")
		if file != "" {
			f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, fmt.Errorf("Unable to open log file for writing: %s", err)
			}

			l.lError = log.New(io.MultiWriter(os.Stderr, f), logPrefixError, log.LstdFlags)
		} else {
			l.lError = log.New(os.Stderr, logPrefixError, log.LstdFlags)
		}

	case LLInfo:
		fmt.Println(logPrefixInfo + "Logging enabled")
		if file != "" {
			f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, fmt.Errorf("Unable to open log file for writing: %s", err)
			}

			l.lError = log.New(io.MultiWriter(os.Stderr, f), logPrefixError, log.LstdFlags)
			l.lInfo = log.New(io.MultiWriter(os.Stdout, f), logPrefixInfo, log.LstdFlags)
		} else {
			l.lError = log.New(os.Stderr, logPrefixError, log.LstdFlags)
			l.lInfo = log.New(os.Stdout, logPrefixInfo, log.LstdFlags)
		}

	case LLDebug:
		fmt.Println(logPrefixDebug + "Logging enabled")

		if file != "" {
			f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, fmt.Errorf("Unable to open log file for writing: %s", err)
			}

			l.lError = log.New(io.MultiWriter(os.Stderr, f), logPrefixError, log.LstdFlags)
			l.lInfo = log.New(io.MultiWriter(os.Stdout, f), logPrefixInfo, log.LstdFlags)
			l.lDebug = log.New(io.MultiWriter(os.Stdout, f), logPrefixDebug, log.LstdFlags)
		} else {
			l.lError = log.New(os.Stderr, logPrefixError, log.LstdFlags)
			l.lInfo = log.New(os.Stdout, logPrefixInfo, log.LstdFlags)
			l.lDebug = log.New(os.Stdout, logPrefixDebug, log.LstdFlags)
		}

	default:
		return nil, fmt.Errorf("Invalid log level: %q", level)
	}

	if file != "" {
		l.Info("Logging to file " + file)
	} else {
		l.Info("Logging to console only")
	}

	return l, nil
}
