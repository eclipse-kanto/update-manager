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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/eclipse-kanto/update-manager/logger"
	"github.com/pkg/errors"
)

func createDefaultTLSConfig(skipVerify bool) *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
		CipherSuites:       supportedCipherSuites(),
		InsecureSkipVerify: skipVerify,
	}
}

// NewTLSConfig initializes the TLS.
func NewTLSConfig(settings *internalConnectionConfig) (*tls.Config, error) {
	if err := validateTLSConfig(settings); err != nil {
		return nil, errors.Wrap(err, "invalid TLS configuration provided")
	}

	// load CA cert
	caCert, err := ioutil.ReadFile(settings.RootCA)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load CA")
	}
	tlsConfig := createDefaultTLSConfig(false)
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, errors.Errorf("failed to parse CA %s", settings.RootCA)
	}
	tlsConfig.RootCAs = caCertPool

	// load client certificate-key pair
	cert, err := tls.LoadX509KeyPair(settings.ClientCert, settings.ClientKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load X509 key pair")
	}
	tlsConfig.Certificates = append(tlsConfig.Certificates, cert)

	return tlsConfig, nil
}

func supportedCipherSuites() []uint16 {
	cs := tls.CipherSuites()
	cid := make([]uint16, len(cs))
	for i := range cs {
		cid[i] = cs[i].ID
	}
	return cid
}

func validateTLSConfig(config *internalConnectionConfig) error {
	if err := validateTLSConfigFile(config.RootCA, ".crt"); err != nil {
		logger.ErrorErr(err, "problem accessing provided CA file %s", config.RootCA)
		return err
	}
	if err := validateTLSConfigFile(config.ClientCert, ".cert"); err != nil {
		logger.ErrorErr(err, "problem accessing provided certificate file %s", config.ClientCert)
		return err
	}
	if err := validateTLSConfigFile(config.ClientKey, ".key"); err != nil {
		logger.ErrorErr(err, "problem accessing provided certificate key file %s", config.ClientKey)
		return err
	}
	return nil
}

func validateTLSConfigFile(file, expectedFileExt string) error {
	if file == "" {
		return newErrorf("TLS configuration data is missing, file must be %s", expectedFileExt)
	}
	if !filepath.IsAbs(file) {
		return newErrorf("provided path must be absolute - %s", file)
	}
	if err := fileNotExistEmptyOrDir(file); err != nil {
		return err
	}
	return nil
}

func fileNotExistEmptyOrDir(filename string) error {
	fi, err := os.Stat(filename)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return newErrorf("the provided path %s is a dir path - file is required", filename)
	}
	if fi.Size() == 0 {
		return newErrorf("file %s is empty", filename)
	}
	return nil
}

func newErrorf(format string, args ...interface{}) error {
	return errors.New(fmt.Sprintf(format, args...))
}
