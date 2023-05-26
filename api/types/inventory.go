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

import "github.com/pkg/errors"

// Inventory defines the payload holding an inventory graph.
type Inventory struct {
	HardwareNodes []*HardwareNode `json:"hardwareNodes,omitempty"`
	SoftwareNodes []*SoftwareNode `json:"softwareNodes,omitempty"`
	Associations  []*Association  `json:"associations,omitempty"`
}

// InventoryNode defines common struct for both SoftwareNode and HardwareNode.
type InventoryNode struct {
	ID         string          `json:"id,omitempty"`
	Version    string          `json:"version,omitempty"`
	Name       string          `json:"name,omitempty"`
	Parameters []*KeyValuePair `json:"parameters,omitempty"`
}

// HardwareNode defines the representation for a hardware node.
type HardwareNode struct {
	InventoryNode
	Addressable bool `json:"addressable,omitempty"`
}

// SoftwareNode defines the representation for a software node.
type SoftwareNode struct {
	InventoryNode
	Type SoftwareType `json:"type,omitempty"`
}

// Association links software and/or hardware nodes.
type Association struct {
	SourceID string `json:"sourceId,omitempty"`
	TargetID string `json:"targetId,omitempty"`
}

// SoftwareType represents the type of a software node.
type SoftwareType string

const (
	// SoftwareTypeImage represents an image software type
	SoftwareTypeImage SoftwareType = "IMAGE"
	// SoftwareTypeRaw represents a raw bytes software type
	SoftwareTypeRaw SoftwareType = "RAW"
	// SoftwareTypeData represents a data software type
	SoftwareTypeData SoftwareType = "DATA"
	// SoftwareTypeApplication represents an application software type
	SoftwareTypeApplication SoftwareType = "APPLICATION"
	// SoftwareTypeContainer represents a container software type
	SoftwareTypeContainer SoftwareType = "CONTAINER"
)

// FromCurrentStateBytes receives Envelope as raw bytes and converts them to Inventory instance.
func FromCurrentStateBytes(bytes []byte) (string, *Inventory, error) {
	payloadCurrentState := &Inventory{}
	envelope, err := FromEnvelope(bytes, payloadCurrentState)
	if err != nil {
		return "", nil, errors.Wrap(err, "cannot unmarshal current state")
	}
	return envelope.ActivityID, payloadCurrentState, nil
}

// FromCurrentStateGetBytes receives Envelope as raw bytes and converts them and returns the activityId.
func FromCurrentStateGetBytes(bytes []byte) (string, error) {
	envelope, err := FromEnvelope(bytes, nil)
	if err != nil {
		return "", errors.Wrap(err, "cannot unmarshal current state get")
	}
	return envelope.ActivityID, nil
}

// ToCurrentStateBytes returns the Envelope as raw bytes, setting activity ID and payload to the given parameters.
func ToCurrentStateBytes(activityID string, currentState *Inventory) ([]byte, error) {
	bytes, err := ToEnvelope(activityID, currentState)
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal current state")
	}
	return bytes, nil
}

// ToCurrentStateGetBytes returns the Envelope as raw bytes, setting activity ID and payload to the given parameters.
func ToCurrentStateGetBytes(activityID string) ([]byte, error) {
	bytes, err := ToEnvelope(activityID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal current state get")
	}
	return bytes, nil
}
