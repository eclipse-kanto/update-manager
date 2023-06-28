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

package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToCurrentStateBytes(t *testing.T) {
	currentState := &Inventory{
		HardwareNodes: []*HardwareNode{
			{
				InventoryNode: InventoryNode{
					ID:      "hardware-node-id",
					Version: "1.0.0",
					Name:    "Hardware Node",
					Parameters: []*KeyValuePair{
						{
							Key:   "x",
							Value: "y",
						},
					},
				},
				Addressable: true,
			},
		},
		SoftwareNodes: []*SoftwareNode{
			{
				InventoryNode: InventoryNode{
					ID:      "software-node-id",
					Version: "1.0.0",
					Name:    "Software Node",
					Parameters: []*KeyValuePair{
						{
							Key:   "x",
							Value: "y",
						},
					},
				},
				Type: SoftwareTypeApplication,
			},
		},
		Associations: []*Association{
			{
				SourceID: "hardware-node-id",
				TargetID: "software-node-id",
			},
		},
	}
	currentStateBytes, err := ToEnvelope("", currentState)
	assert.NoError(t, err)

	currentStateEnvelope := &Envelope{
		Payload: &Inventory{},
	}
	assert.NoError(t, json.Unmarshal(currentStateBytes, currentStateEnvelope))

	assert.Empty(t, currentStateEnvelope.ActivityID)
	assert.Equal(t, currentState, currentStateEnvelope.Payload)
}
