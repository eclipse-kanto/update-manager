// Copyright (c) 2023 Contributors to the Eclipse Foundation
//
// See the NOTICE file(s) distributed with this work for additional
// information regarding copyright ownership.
//
// This program and the accompanying materials are made available under the
// terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0, or the Apache License, Version 2.0
// which is available at https://www.apache.org/licenses/LICENSE-2.0.
//
// SPDX-License-Identifier: EPL-2.0 OR Apache-2.0

package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LogConfig contains logging configuration
type LogConfig struct {
	LogFile       string `json:"logFile,omitempty"`
	LogLevel      string `json:"logLevel,omitempty"`
	LogFileSize   int    `json:"logFileSize,omitempty"`
	LogFileCount  int    `json:"logFileCount,omitempty"`
	LogFileMaxAge int    `json:"logFileMaxAge,omitempty"`
}

// LogLevel - Error(1), Warn(2), Info(3), Debug(4) or Trace(5)
type LogLevel int

// Constants for log level
const (
	ERROR LogLevel = 1 + iota
	WARN
	INFO
	DEBUG
	TRACE
)

const (
	logFlags int = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lmsgprefix

	ePrefix = "ERROR  "
	wPrefix = "WARN   "
	iPrefix = "INFO   "
	dPrefix = "DEBUG  "
	tPrefix = "TRACE  "

	prefix = " %s "
)

var (
	logger *log.Logger
	level  LogLevel
)

// SetupLogger initializes logger with the provided configuration
func SetupLogger(logConfig *LogConfig, componentPrefix string) (io.WriteCloser, error) {
	loggerOut := io.WriteCloser(&nopWriterCloser{out: os.Stderr})
	if len(logConfig.LogFile) > 0 {
		err := os.MkdirAll(filepath.Dir(logConfig.LogFile), 0755)

		if err != nil {
			return nil, err
		}

		loggerOut = &lumberjack.Logger{
			Filename:   logConfig.LogFile,
			MaxSize:    logConfig.LogFileSize,
			MaxBackups: logConfig.LogFileCount,
			MaxAge:     logConfig.LogFileMaxAge,
			LocalTime:  true,
			Compress:   true,
		}
	}

	log.SetOutput(loggerOut)
	log.SetFlags(logFlags)

	logger = log.New(loggerOut, fmt.Sprintf(prefix, componentPrefix), logFlags)

	// Parse log level
	switch strings.ToUpper(logConfig.LogLevel) {
	case "INFO":
		level = INFO
	case "WARN":
		level = WARN
	case "DEBUG":
		level = DEBUG
	case "TRACE":
		level = TRACE
	default:
		level = ERROR
	}

	return loggerOut, nil
}

// Error logs the given formatted message and value, if level is >= ERROR
func Error(format string, v ...interface{}) {
	if level >= ERROR {
		logger.Println(fmt.Errorf(fmt.Sprint(ePrefix, " ", format), v...))
	}
}

// ErrorErr logs the given value, formatted message and error, if level is >= ERROR
func ErrorErr(err error, format string, v ...interface{}) {
	if level >= ERROR {
		logger.Println(fmt.Errorf(fmt.Sprint(ePrefix, " ", format, " ", err), v...))
	}
}

// Warn logs the given formatted message and value, if level is >= WARN
func Warn(format string, v ...interface{}) {
	if level >= WARN {
		logger.Printf(fmt.Sprint(wPrefix, " ", format), v...)
	}
}

// WarnErr logs the given value, formatted message and error, if level is >= WARN
func WarnErr(err error, format string, v ...interface{}) {
	if level >= WARN {
		logger.Println(fmt.Errorf(fmt.Sprint(wPrefix, " ", format, " ", err), v...))
	}
}

// Info logs the given formatted message and value, if level is >= INFO
func Info(format string, v ...interface{}) {
	if level >= INFO {
		logger.Printf(fmt.Sprint(iPrefix, " ", format), v...)
	}
}

// InfoErr logs the given value, formatted message and error, if level is >= INFO
func InfoErr(err error, format string, v ...interface{}) {
	if level >= INFO {
		logger.Println(fmt.Errorf(fmt.Sprint(iPrefix, " ", format, " ", err), v...))
	}
}

// Debug logs the given formatted message and value, if level is >= DEBUG
func Debug(format string, v ...interface{}) {
	if IsDebugEnabled() {
		logger.Printf(fmt.Sprint(dPrefix, " ", format), v...)
	}
}

// DebugErr logs the given value, formatted message and error, if level is >= DEBUG
func DebugErr(err error, format string, v ...interface{}) {
	if IsDebugEnabled() {
		logger.Println(fmt.Errorf(fmt.Sprint(dPrefix, " ", format, " ", err), v...))
	}
}

// Trace logs the given formatted message and value, if level is >= TRACE
func Trace(format string, v ...interface{}) {
	if IsTraceEnabled() {
		logger.Printf(fmt.Sprint(tPrefix, " ", format), v...)
	}
}

// TraceErr logs the given value, formatted message and error, if level is >= TRACE
func TraceErr(err error, format string, v ...interface{}) {
	if IsTraceEnabled() {
		logger.Println(fmt.Errorf(fmt.Sprint(tPrefix, " ", format, " ", err), v...))
	}
}

// IsDebugEnabled returns true if log level is above DEBUG
func IsDebugEnabled() bool {
	return level >= DEBUG
}

// IsTraceEnabled returns true if log level is above TRACE
func IsTraceEnabled() bool {
	return level >= TRACE
}

type nopWriterCloser struct {
	out io.Writer
}

// Write to log output
func (w *nopWriterCloser) Write(p []byte) (n int, err error) {
	return w.out.Write(p)
}

// Close does nothing
func (*nopWriterCloser) Close() error {
	return nil
}
