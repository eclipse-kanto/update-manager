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

package types

// ConsentStatusType defines values for status within the owner consent
type ConsentStatusType string

const (
	// StatusApproved denotes that the owner has consented.
	StatusApproved ConsentStatusType = "APPROVED"
	// StatusDenied denotes that the owner has not consented.
	StatusDenied ConsentStatusType = "DENIED"
)

// OwnerConsent defines the payload for Owner Consent response.
type OwnerConsent struct {
	Status ConsentStatusType `json:"status,omitempty"`
	// time field for scheduling could be added here
}
