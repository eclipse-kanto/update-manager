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
	defaultBroker             = "tcp://localhost:1883"
	defaultKeepAlive          = "20s"
	defaultDisconnectTimeout  = "250ms"
	defaultUsername           = ""
	defaultPassword           = ""
	defaultConnectTimeout     = "30s"
	defaultAcknowledgeTimeout = "15s"
	defaultSubscribeTimeout   = "15s"
	defaultUnsubscribeTimeout = "5s"
	defaultCACert             = ""
	defaultCert               = ""
	defaultKey                = ""
)

// ConnectionConfig represents the mqtt client connection config
type ConnectionConfig struct {
	Broker             string `json:"broker,omitempty"`
	KeepAlive          string `json:"keepAlive,omitempty"`
	DisconnectTimeout  string `json:"disconnectTimeout,omitempty"`
	Username           string `json:"username,omitempty"`
	Password           string `json:"password,omitempty"`
	ConnectTimeout     string `json:"connectTimeout,omitempty"`
	AcknowledgeTimeout string `json:"acknowledgeTimeout,omitempty"`
	SubscribeTimeout   string `json:"subscribeTimeout,omitempty"`
	UnsubscribeTimeout string `json:"unsubscribeTimeout,omitempty"`
	CACert             string `json:"caCert"`
	Cert               string `json:"cert"`
	Key                string `json:"key"`
}

// NewDefaultConfig returns a default mqtt client connection config instance
func NewDefaultConfig() *ConnectionConfig {
	return &ConnectionConfig{
		Broker:             defaultBroker,
		KeepAlive:          defaultKeepAlive,
		DisconnectTimeout:  defaultDisconnectTimeout,
		Username:           defaultUsername,
		Password:           defaultPassword,
		ConnectTimeout:     defaultConnectTimeout,
		AcknowledgeTimeout: defaultAcknowledgeTimeout,
		SubscribeTimeout:   defaultSubscribeTimeout,
		UnsubscribeTimeout: defaultUnsubscribeTimeout,
		CACert:             defaultCACert,
		Cert:               defaultCert,
		Key:                defaultKey,
	}
}
