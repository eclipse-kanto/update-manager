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

package mqtt

const (
	// default mqtt connection config
	defaultBrokerURL          = "tcp://localhost:1883"
	defaultKeepAlive          = 20000
	defaultDisconnectTimeout  = 250
	defaultClientUsername     = ""
	defaultClientPassword     = ""
	defaultConnectTimeout     = 30000
	defaultAcknowledgeTimeout = 15000
	defaultSubscribeTimeout   = 15000
	defaultUnsubscribeTimeout = 5000
)

// ConnectionConfig represents the mqtt client connection config
type ConnectionConfig struct {
	BrokerURL          string `json:"brokerUrl,omitempty"`
	KeepAlive          int64  `json:"keepAlive,omitempty"`
	DisconnectTimeout  int64  `json:"disconnectTimeout,omitempty"`
	ClientUsername     string `json:"clientUsername,omitempty"`
	ClientPassword     string `json:"clientPassword,omitempty"`
	ConnectTimeout     int64  `json:"connectTimeout,omitempty"`
	AcknowledgeTimeout int64  `json:"acknowledgeTimeout,omitempty"`
	SubscribeTimeout   int64  `json:"subscribeTimeout,omitempty"`
	UnsubscribeTimeout int64  `json:"unsubscribeTimeout,omitempty"`
}

// NewDefaultConfig returns a default mqtt client connection config instance
func NewDefaultConfig() *ConnectionConfig {
	return &ConnectionConfig{
		BrokerURL:          defaultBrokerURL,
		KeepAlive:          defaultKeepAlive,
		DisconnectTimeout:  defaultDisconnectTimeout,
		ClientUsername:     defaultClientUsername,
		ClientPassword:     defaultClientPassword,
		ConnectTimeout:     defaultConnectTimeout,
		AcknowledgeTimeout: defaultAcknowledgeTimeout,
		SubscribeTimeout:   defaultSubscribeTimeout,
		UnsubscribeTimeout: defaultUnsubscribeTimeout,
	}
}
