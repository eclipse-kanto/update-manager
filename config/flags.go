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

package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/eclipse-kanto/update-manager/logger"
)

const (
	// config file flag
	configFileFlagID = "config-file"
)

// SetupFlags adds common flags for the configuration of all update agents
func SetupFlags(flagSet *flag.FlagSet, cfg *BaseConfig) {
	flagSet.String(configFileFlagID, "", "Specify the configuration file")

	// init log flags
	flagSet.StringVar(&cfg.Log.LogLevel, "log-level", EnvToString("LOG_LEVEL", cfg.Log.LogLevel), "Set the log level - possible values are ERROR, WARN, INFO, DEBUG, TRACE")
	flagSet.StringVar(&cfg.Log.LogFile, "log-file", EnvToString("LOG_FILE", cfg.Log.LogFile), "Set the log file")
	flagSet.IntVar(&cfg.Log.LogFileSize, "log-file-size", int(EnvToInt("LOG_FILE_SIZE", int64(cfg.Log.LogFileSize))), "Set the maximum size in megabytes of the log file before it gets rotated")
	flagSet.IntVar(&cfg.Log.LogFileCount, "log-file-count", int(EnvToInt("LOG_FILE_COUNT", int64(cfg.Log.LogFileCount))), "Set the maximum number of old log files to retain")
	flagSet.IntVar(&cfg.Log.LogFileMaxAge, "log-file-max-age", int(EnvToInt("LOG_FILE_MAX_AGE", int64(cfg.Log.LogFileMaxAge))), "Set the maximum number of days to retain old log files based on the timestamp encoded in their filename")

	// init connection flags
	flagSet.StringVar(&cfg.MQTT.Broker, "mqtt-conn-broker", EnvToString("MQTT_CONN_BROKER", cfg.MQTT.Broker), "Address of the MQTT server/broker that the update manager will connect for the communication, the format is: scheme://host:port")
	flagSet.StringVar(&cfg.MQTT.KeepAlive, "mqtt-conn-keep-alive", EnvToString("MQTT_CONN_KEEP_ALIVE", cfg.MQTT.KeepAlive), "Keep alive duration for the MQTT requests as duration string")
	flagSet.StringVar(&cfg.MQTT.DisconnectTimeout, "mqtt-conn-disconnect-timeout", EnvToString("MQTT_CONN_DISCONNECT_TIMEOUT", cfg.MQTT.DisconnectTimeout), "Disconnect timeout for the MQTT server/broker as duration string")
	flagSet.StringVar(&cfg.MQTT.Username, "mqtt-conn-username", EnvToString("MQTT_CONN_USERNAME", cfg.MQTT.Username), "Username that is a part of the credentials")
	flagSet.StringVar(&cfg.MQTT.Password, "mqtt-conn-password", EnvToString("MQTT_CONN_PASSWORD", cfg.MQTT.Password), "Password that is a part of the credentials")
	flagSet.StringVar(&cfg.MQTT.ConnectTimeout, "mqtt-conn-connect-timeout", EnvToString("MQTT_CONN_CONNECT_TIMEOUT", cfg.MQTT.ConnectTimeout), "Connect timeout for the MQTT server/broker as duration string")
	flagSet.StringVar(&cfg.MQTT.AcknowledgeTimeout, "mqtt-conn-ack-timeout", EnvToString("MQTT_CONN_ACK_TIMEOUT", cfg.MQTT.AcknowledgeTimeout), "Acknowledge timeout for the MQTT requests as duration string")
	flagSet.StringVar(&cfg.MQTT.SubscribeTimeout, "mqtt-conn-sub-timeout", EnvToString("MQTT_CONN_SUB_TIMEOUT", cfg.MQTT.SubscribeTimeout), "Subscribe timeout for the MQTT requests as duration string")
	flagSet.StringVar(&cfg.MQTT.UnsubscribeTimeout, "mqtt-conn-unsub-timeout", EnvToString("MQTT_CONN_UNSUB_TIMEOUT", cfg.MQTT.UnsubscribeTimeout), "Unsubscribe timeout for the MQTT requests as duration string")

	flagSet.StringVar(&cfg.Domain, "domain", EnvToString("DOMAIN", cfg.Domain), "Specify the Domain of this update agent, used as MQTT topic prefix.")

	flagSet.BoolVar(&cfg.ThingsEnabled, "things-enabled", EnvToBool("THINGS_ENABLED", cfg.ThingsEnabled), "Specify whether the UpdateManager will behave as a things.")
}

// ParseConfigFilePath returns the value for configuration file path if set.
func ParseConfigFilePath() string {
	var cfgFilePath string
	flagSet := flag.NewFlagSet("", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	flagSet.StringVar(&cfgFilePath, configFileFlagID, "", "Specify the configuration file")
	if err := flagSet.Parse(getFlagArgs(configFileFlagID)); err != nil {
		logger.ErrorErr(err, "Cannot parse the config-file flag")
	}
	return cfgFilePath
}

// EnvToString check if an ENV variable is set and returns its value as a string. If not set, the default value is returned.
func EnvToString(key string, value string) string {
	envVal, ok := os.LookupEnv(key)
	if !ok {
		return value
	}
	fmt.Printf("using ENV variable %s with value %s\n", key, envVal)
	return envVal
}

// EnvToBool check if an ENV variable is set and returns its value as a bool. If not set, the default value is returned.
func EnvToBool(key string, value bool) bool {
	envVal, ok := os.LookupEnv(key)
	if !ok {
		return value
	}
	boolVal, err := strconv.ParseBool(envVal)
	if err != nil {
		fmt.Printf("cannot use ENV variable %s with value %s\n", key, envVal)
		return value
	}
	fmt.Printf("using ENV variable %s with value %s\n", key, envVal)
	return boolVal
}

// EnvToInt check if an ENV variable is set and returns its value as an integer. If not set or value is not an integer, the default value is returned.
func EnvToInt(key string, value int64) int64 {
	envVal, ok := os.LookupEnv(key)
	if !ok {
		return value
	}
	intVal, err := strconv.ParseInt(envVal, 10, 0)
	if err != nil {
		fmt.Printf("cannot use ENV variable %s with value %s\n", key, envVal)
		return value
	}
	fmt.Printf("using ENV variable %s with value %s\n", key, envVal)
	return intVal
}
