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
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToEnvelope(t *testing.T) {
	testActivityID := "testActivityID"
	testPayload := &Inventory{
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
	}

	expected, err1 := json.Marshal(&Envelope{
		ActivityID: testActivityID,
		Timestamp:  time.Now().UnixNano() / int64(time.Millisecond),
		Payload:    testPayload,
	})
	assert.NoError(t, err1)

	actual, err := ToEnvelope(testActivityID, testPayload)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestFromEnvelope(t *testing.T) {
	testPayload := &Inventory{
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
	}

	t.Run("test_from_envelope_ok", func(t *testing.T) {
		payloadToBytes, err := json.Marshal(testPayload)
		assert.NoError(t, err)
		expected := &Envelope{
			Payload: testPayload,
		}
		actual, err := FromEnvelope(payloadToBytes, testPayload)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("test_from_envelope_err", func(t *testing.T) {
		_, err := FromEnvelope([]byte{}, testPayload)
		assert.Error(t, err)
	})
}
