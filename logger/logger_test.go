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
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLogLevelError tests logger functions with log level set to ERROR.
func TestLogLevelError(t *testing.T) {
	validate("ERROR", true, false, false, false, false, t)
}

// TestLogLevelWarn tests logger functions with log level set to WARN.
func TestLogLevelWarn(t *testing.T) {
	validate("WARN", true, true, false, false, false, t)
}

// TestLogLevelInfo tests logger functions with log level set to INFO.
func TestLogLevelInfo(t *testing.T) {
	validate("INFO", true, true, true, false, false, t)
}

// TestLogLevelDebug tests logger functions with log level set to DEBUG.
func TestLogLevelDebug(t *testing.T) {
	validate("DEBUG", true, true, true, true, false, t)
}

// TestLogLevelTrace tests logger functions with log level set to TRACE.
func TestLogLevelTrace(t *testing.T) {
	validate("TRACE", true, true, true, true, true, t)
}

// TestNopWriter tests logger functions without writer.
func TestNopWriter(t *testing.T) {
	// Prepare
	dir := "_tmp-logger"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	// Prepare the logger without writer
	loggerOut, _ := SetupLogger(&LogConfig{LogFile: "", LogLevel: "TRACE", LogFileSize: 2, LogFileCount: 5}, "[logger-test]")
	defer loggerOut.Close()

	// Validate that temporary is empty
	Error("test error")
	f, err := os.Open(dir)
	if err != nil {
		t.Fatalf("cannot open temporary directory: %v", err)
	}
	defer f.Close()

	if _, err = f.Readdirnames(1); err != io.EOF {
		t.Errorf("temporary directory is not empty")
	}
}

func validate(lvl string, hasError bool, hasWarn bool, hasInfo bool, hasDebug bool, hasTrace bool, t *testing.T) {
	// Prepare
	dir := "_tmp-logger"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	// Prepare the logger
	log := filepath.Join(dir, lvl+".log")
	loggerOut, err := SetupLogger(&LogConfig{LogFile: log, LogLevel: lvl, LogFileSize: 2, LogFileCount: 5}, "[logger-test]")
	if err != nil {
		t.Fatal(err)
	}
	defer loggerOut.Close()

	// 1. Validate for error logs.
	validateError(log, hasError, t)

	// 2. Validate for warn logs.
	validateWarn(log, hasWarn, t)

	// 3. Validate for info logs.
	validateInfo(log, hasInfo, t)

	// 4. Validate for debug logs.
	validateDebug(log, hasDebug, t)

	// 5. Validate for trace logs.
	validateTrace(log, hasTrace, t)

}

// validateError validates for error logs.
func validateError(log string, has bool, t *testing.T) {
	// 1. Validate for Error function wihout additional parameters.
	Error("error log without parameters")
	if has != search(log, t, ePrefix, "error log without parameters") {
		t.Errorf("error entry mismatch [result: %v]", !has)
	}
	// 2. Validate for Error function with additional parameters.
	Error("error log with parameters [%s,%s]", "param1", "param2")
	if has != search(log, t, ePrefix, "error log with parameters [param1,param2]") {
		t.Errorf("error entry mismatch [result: %v]", !has)
	}
	// 3. Validate for ErrorErr function without additional parameters.
	testErrorErr := fmt.Errorf("testErrorErr")
	ErrorErr(testErrorErr, "errorErr log without parameters")
	if has != search(log, t, ePrefix, "errorErr log without parameters testErrorErr") {
		t.Errorf("errorErr entry mismatch: [result: %v]", !has)
	}
	// 4. Validate for ErrorErr function without additional parameters
	ErrorErr(testErrorErr, "errorErr log with parameters [%s,%s]", "param1", "param2")
	if has != search(log, t, ePrefix, "errorErr log with parameters [param1,param2] testErrorErr") {
		t.Errorf("errorErr entry mismatch: [result: %v]", !has)
	}
}

// validateWarn validates for warn logs.
func validateWarn(log string, has bool, t *testing.T) {
	// 1. Validate for Warn function wihout additional parameters.
	Warn("warn log without parameters")
	if has != search(log, t, wPrefix, "warn log without parameters") {
		t.Errorf("warn entry mismatch [result: %v]", !has)
	}
	// 2. Validate for Warn function with additional parameters.
	Warn("warn log with parameters [%s,%s]", "param1", "param2")
	if has != search(log, t, wPrefix, "warn log with parameters [param1,param2]") {
		t.Errorf("warn entry mismatch [result: %v]", !has)
	}
	// 3. Validate for WarnErr function without additional parameters.
	testWarnErr := fmt.Errorf("testWarnErr")
	WarnErr(testWarnErr, "warnErr log without parameters")
	if has != search(log, t, wPrefix, "warnErr log without parameters testWarnErr") {
		t.Errorf("warnErr entry mismatch: [result: %v]", !has)
	}
	// 4. Validate for WarnErr function without additional parameters
	WarnErr(testWarnErr, "warnErr log with parameters [%s,%s]", "param1", "param2")
	if has != search(log, t, wPrefix, "warnErr log with parameters [param1,param2] testWarnErr") {
		t.Errorf("warnErr entry mismatch: [result: %v]", !has)
	}
}

// validateInfo validates for info logs.
func validateInfo(log string, has bool, t *testing.T) {
	// 1. Validate for Info function wihout additional parameters.
	Info("info log without parameters")
	if has != search(log, t, iPrefix, "info log without parameters") {
		t.Errorf("info entry mismatch [result: %v]", !has)
	}
	// 2. Validate for Info function with additional parameters.
	Info("info log with parameters [%s,%s]", "param1", "param2")
	if has != search(log, t, iPrefix, "info log with parameters [param1,param2]") {
		t.Errorf("info entry mismatch [result: %v]", !has)
	}
	// 3. Validate for InfoErr function without additional parameters.
	testInfoErr := fmt.Errorf("testInfoErr")
	InfoErr(testInfoErr, "infoErr log without parameters")
	if has != search(log, t, iPrefix, "infoErr log without parameters testInfoErr") {
		t.Errorf("infoErr entry mismatch: [result: %v]", !has)
	}
	// 4. Validate for InfoErr function with additional parameters
	InfoErr(testInfoErr, "infoErr log with parameters [%s,%s]", "param1", "param2")
	if has != search(log, t, iPrefix, "infoErr log with parameters [param1,param2] testInfoErr") {
		t.Errorf("infoErr entry mismatch: [result: %v]", !has)
	}
}

// validateDebug validates for debug logs.
func validateDebug(log string, has bool, t *testing.T) {
	// 1. Validate for Debug function wihout additional parameters.
	Debug("debug log without parameters")
	if has != search(log, t, dPrefix, "debug log without parameters") {
		t.Errorf("debug entry mismatch [result: %v]", !has)
	}
	// 2. Validate for Debug function with additional parameters.
	Debug("debug log with parameters [%s,%s]", "param1", "param2")
	if has != search(log, t, dPrefix, "debug log with parameters [param1,param2]") {
		t.Errorf("debug entry mismatch [result: %v]", !has)
	}
	// 3. Validate for DebugErr function without additional parameters.
	testDebugErr := fmt.Errorf("testDebugErr")
	DebugErr(testDebugErr, "debugErr log without parameters")
	if has != search(log, t, dPrefix, "debugErr log without parameters testDebugErr") {
		t.Errorf("debugErr entry mismatch: [result: %v]", !has)
	}
	// 4. Validate for DebugErr function with additional parameters
	DebugErr(testDebugErr, "debugErr log with parameters [%s,%s]", "param1", "param2")
	if has != search(log, t, dPrefix, "debugErr log with parameters [param1,param2] testDebugErr") {
		t.Errorf("debugErr entry mismatch: [result: %v]", !has)
	}
}

// validateTrace validates for trace logs.
func validateTrace(log string, has bool, t *testing.T) {
	// 1. Validate for Trace function wihout additional parameters.
	Trace("trace log without parameters")
	if has != search(log, t, tPrefix, "trace log without parameters") {
		t.Errorf("trace entry mismatch [result: %v]", !has)
	}
	// 2. Validate for Trace function with additional parameters.
	Trace("trace log with parameters [%s,%s]", "param1", "param2")
	if has != search(log, t, tPrefix, "trace log with parameters [param1,param2]") {
		t.Errorf("trace entry mismatch [result: %v]", !has)
	}
	// 3. Validate for TraceErr function without additional parameters.
	testTraceErr := fmt.Errorf("testTraceErr")
	TraceErr(testTraceErr, "traceErr log without parameters")
	if has != search(log, t, tPrefix, "traceErr log without parameters testTraceErr") {
		t.Errorf("traceErr entry mismatch: [result: %v]", !has)
	}
	// 4. Validate for TraceErr function with additional parameters
	TraceErr(testTraceErr, "traceErr log with parameters [%s,%s]", "param1", "param2")
	if has != search(log, t, tPrefix, "traceErr log with parameters [param1,param2] testTraceErr") {
		t.Errorf("traceErr entry mismatch: [result: %v]", !has)
	}
}

// search strings in log file.
func search(fn string, t *testing.T, entries ...string) bool {
	file, err := os.Open(fn)
	if err != nil {
		t.Fatalf("fail to open log file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if has(scanner.Text(), entries...) {
			return true
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("fail to read log file: %v", err)
	}
	return false
}

// has checks if string has substrings
func has(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if !strings.Contains(s, substr) {
			return false
		}
	}
	return true
}
