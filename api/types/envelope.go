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
	"time"
)

// Envelope defines the structure of messages for interacting with the UpdateAgent API.
type Envelope struct {
	ActivityID string      `json:"activityId"`
	Timestamp  int64       `json:"timestamp"`
	Payload    interface{} `json:"payload"`
}

// FromEnvelope expects to receive raw bytes and convert them as Envelope, payload being unmarshalled into the given parameter.
func FromEnvelope(bytes []byte, payload interface{}) (*Envelope, error) {
	envelope := &Envelope{
		Payload: payload,
	}
	if err := json.Unmarshal(bytes, envelope); err != nil {
		return nil, err
	}
	return envelope, nil
}

// ToEnvelope returns the Envelope as raw bytes, setting activity ID and payload to the given parameters.
func ToEnvelope(activityID string, payload interface{}) ([]byte, error) {
	envelope := &Envelope{
		ActivityID: activityID,
		Timestamp:  time.Now().UnixNano() / int64(time.Millisecond),
		Payload:    payload,
	}
	return json.Marshal(envelope)
}
