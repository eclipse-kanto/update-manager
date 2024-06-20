// Copyright (c) 2024 Contributors to the Eclipse Foundation
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
	"fmt"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
)

type ownerConsentThingsClient struct {
	*updateAgentThingsClient
	domain string
}

// NewOwnerConsentThingsClient instantiates a new client for handling owner consent approval through things.
func NewOwnerConsentThingsClient(domain string, updateAgent api.UpdateAgentClient) (api.OwnerConsentClient, error) {
	var client *updateAgentThingsClient
	switch v := updateAgent.(type) {
	case *updateAgentThingsClient:
		client = updateAgent.(*updateAgentThingsClient)
	default:
		return nil, fmt.Errorf("unexpected type: %T", v)
	}
	return &ownerConsentThingsClient{
		updateAgentThingsClient: client,
		domain:                  domain,
	}, nil
}

func (client *ownerConsentThingsClient) Domain() string {
	return client.domain
}

// Start sets consent handler to the UpdateManager feature.
func (client *ownerConsentThingsClient) Start(consentHandler api.OwnerConsentHandler) error {
	client.umFeature.SetConsentHandler(consentHandler)
	return nil
}

// Stop removes the consent handler from the UpdateManager feature.
func (client *ownerConsentThingsClient) Stop() error {
	client.umFeature.SetConsentHandler(nil)
	return nil
}

// SendOwnerConsent requests the owner consent through the UpdateManager feature.
func (client *ownerConsentThingsClient) SendOwnerConsent(activityID string, consent *types.OwnerConsent) error {
	return client.umFeature.SendConsent(activityID, consent)
}
