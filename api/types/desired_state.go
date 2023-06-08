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
	"strings"

	"github.com/pkg/errors"
)

// CommandType defines command names for desired state command requests
type CommandType string

const (
	// CommandDownload denotes a download baseline command
	CommandDownload CommandType = "DOWNLOAD"
	// CommandUpdate denotes an update baseline command
	CommandUpdate CommandType = "UPDATE"
	// CommandActivate denotes an activate baseline command
	CommandActivate CommandType = "ACTIVATE"
	// CommandRollback denotes a rollback baseline command
	CommandRollback CommandType = "ROLLBACK"
	// CommandCleanup denotes a cleanup baseline command
	CommandCleanup CommandType = "CLEANUP"
)

// DesiredState defines the payload holding the Desired State specification.
type DesiredState struct {
	Baselines []*Baseline `json:"baselines,omitempty"`
	Domains   []*Domain   `json:"domains,omitempty"`
}

// DesiredStateCommand defines the payload holding the Desired State Command specification.
type DesiredStateCommand struct {
	Command  CommandType `json:"command,omitempty"`
	Baseline string      `json:"baseline,omitempty"`
}

// Baseline defines the baseline structure within the Desired State specification.
type Baseline struct {
	Title         string   `json:"title,omitempty"`
	Description   string   `json:"description,omitempty"`
	Preconditions string   `json:"preconditions,omitempty"`
	Components    []string `json:"components,omitempty"`
}

// Domain defines the domain structure within the Desired State specification.
type Domain struct {
	ID         string                 `json:"id,omitempty"`
	Config     []*KeyValuePair        `json:"config,omitempty"`
	Components []*ComponentWithConfig `json:"components,omitempty"`
}

// Component defines the component structure within the Desired State specification.
type Component struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
}

// ComponentWithConfig defines the component structure within the Desired State specification, it includes key-value configuration pairs.
type ComponentWithConfig struct {
	Component
	Config []*KeyValuePair `json:"config,omitempty"`
}

// KeyValuePair represents key/value string pair.
type KeyValuePair struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// FromDesiredStateBytes receives Envelope as raw bytes and converts them to Desired State instance.
func FromDesiredStateBytes(bytes []byte) (string, *DesiredState, error) {
	payloadState := &DesiredState{}
	envelope, err := FromEnvelope(bytes, payloadState)
	if err != nil {
		return "", nil, errors.Wrap(err, "cannot unmarshal state")
	}
	return envelope.ActivityID, payloadState, nil
}

// FromDesiredStateCommandBytes receives Envelope as raw bytes and converts them to Desired State Command instance.
func FromDesiredStateCommandBytes(bytes []byte) (string, *DesiredStateCommand, error) {
	payloadStateCommand := &DesiredStateCommand{}
	envelope, err := FromEnvelope(bytes, payloadStateCommand)
	if err != nil {
		return "", nil, errors.Wrap(err, "cannot unmarshal state command")
	}
	return envelope.ActivityID, payloadStateCommand, nil
}

// ToDesiredStateCommandBytes returns the Envelope as raw bytes, setting activity ID and payload to the given parameters.
func ToDesiredStateCommandBytes(activityID string, command *DesiredStateCommand) ([]byte, error) {
	bytes, err := ToEnvelope(activityID, command)
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal desired state command")
	}
	return bytes, nil
}

// SplitPerDomains splits the full desired state into several desired states for each domain
func (desiredState *DesiredState) SplitPerDomains() map[string]*DesiredState {
	split := map[string]*DesiredState{}
	for i, domain := range desiredState.Domains {
		split[domain.ID] = &DesiredState{
			Baselines: desiredState.GetBaselinesForDomain(domain.ID),
			Domains:   desiredState.Domains[i : i+1],
		}
	}
	return split
}

// GetBaselinesForDomain returns a list of baselines for the requested domain
func (desiredState *DesiredState) GetBaselinesForDomain(domain string) []*Baseline {
	baselines := []*Baseline{}
	for _, baseline := range desiredState.Baselines {
		baselineComponents := []string{}
		for _, component := range baseline.Components {
			if strings.HasPrefix(component, domain+":") {
				baselineComponents = append(baselineComponents, component)
			}
		}
		if len(baselineComponents) > 0 {
			baseline := &Baseline{
				Title:         baseline.Title,
				Description:   baseline.Description,
				Preconditions: baseline.Preconditions,
				Components:    baselineComponents,
			}
			baselines = append(baselines, baseline)
		}
	}
	if len(baselines) == 0 {
		return nil
	}
	return baselines
}
