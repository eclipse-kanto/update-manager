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

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	notAbsolute = "not-absolute-path"
	caCertPath  = "./testdata/testCaCert.crt"
	certPath    = "./testdata/testCert.cert"
	notAbsError = "invalid TLS configuration provided: provided path must be absolute - " + notAbsolute
	emptyError  = "invalid TLS configuration provided: TLS configuration data is missing"
)

type testTLSConfigError struct {
	config   *internalConnectionConfig
	expError string
}

func TestNewTLSConfigWithError(t *testing.T) {
	nonExisting, _ := filepath.Abs("./nonexisting.test")
	caCertAbsPath, _ := filepath.Abs(caCertPath)
	certAbsPath, _ := filepath.Abs(certPath)
	emptyAbsPath, _ := filepath.Abs("./testdata/emptyTestCertFile.crt")
	dirAbsPath, _ := filepath.Abs("./")

	tests := []testTLSConfigError{
		// missing CACert file
		{
			config:   &internalConnectionConfig{},
			expError: emptyError,
		},
		// CACert must be absolute path
		{
			config:   &internalConnectionConfig{CACert: notAbsolute},
			expError: notAbsError,
		},
		// Cannot find file
		{
			config:   &internalConnectionConfig{CACert: nonExisting},
			expError: "invalid TLS configuration provided: stat " + nonExisting + ": no such file or directory",
		},
		{
			config:   &internalConnectionConfig{CACert: caCertAbsPath},
			expError: emptyError,
		},
		// Cert must be absolute path
		{
			config: &internalConnectionConfig{
				CACert: caCertAbsPath,
				Cert:   notAbsolute,
			},
			expError: notAbsError,
		},
		// Cert file is directory
		{
			config: &internalConnectionConfig{
				CACert: caCertAbsPath,
				Cert:   dirAbsPath,
			},
			expError: "invalid TLS configuration provided: the provided path " + dirAbsPath + " is a dir path - file is required",
		},
		{
			config: &internalConnectionConfig{
				CACert: caCertAbsPath,
				Cert:   certAbsPath,
			},
			expError: emptyError,
		},
		// Key must be absolute path
		{
			config: &internalConnectionConfig{
				CACert: caCertAbsPath,
				Cert:   certAbsPath,
				Key:    notAbsolute,
			},
			expError: notAbsError,
		},
		// Key file is empty
		{
			config: &internalConnectionConfig{
				CACert: caCertAbsPath,
				Cert:   certAbsPath,
				Key:    emptyAbsPath,
			},
			expError: "invalid TLS configuration provided: file " + emptyAbsPath + " is empty",
		},
	}

	for _, test := range tests {
		tlsConfig, err := NewTLSConfig(test.config)
		assert.EqualError(t, err, test.expError)
		assert.Nil(t, tlsConfig)
	}
}

func TestNewTLSConfig(t *testing.T) {
	caCertAbsPath, _ := filepath.Abs(caCertPath)
	certAbsPath, _ := filepath.Abs(certPath)
	keyAbsPath, _ := filepath.Abs("./testdata/testKey.key")

	config := &internalConnectionConfig{
		CACert: caCertAbsPath,
		Cert:   certAbsPath,
		Key:    keyAbsPath,
	}
	tlsConfig, err := NewTLSConfig(config)
	assert.NoError(t, err)
	assert.NotNil(t, tlsConfig)
}
