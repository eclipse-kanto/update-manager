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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromDesiredStateBytes(t *testing.T) {
	spec := `{
		"activityId": "random-activity-id",
		"apiVersion": "1",
		"timestamp": 16656700000000,
		"payload": {
			"baselines": [
				{
					"title": "powertrain 2022",
					"components": [
						"safety-ecu:powertrain",
						"safety-ecu:body"
					]
				},
				{
					"title": "Composite-app123",
					"components": [
						"containers:xyz",
						"safety-app:aap-base-1",
						"safety-app:aap-app-1"
					]
				}
			],
			"domains": [
				{
					"id": "containers",
					"config": [
						{
							"key": "source",
							"value": "my-container-registry.com"
						}
					],
					"components": [
						{
							"id": "xyz",
							"version": "1",
							"config": [
								{
									"key": "key1",
									"value": "val1"
								},
								{
									"key": "key2",
									"value": "val2"
								}
							]
						},
						{
							"id": "abc",
							"version": "4",
							"config": [
								{
									"key": "k1",
									"value": "v1"
								},
								{
									"key": "k2",
									"value": "v2"
								}
							]
						}
					]
				},
				{
					"id": "safety-app",
					"components": [
						{
							"id": "aap-base-1",
							"version": "1.0"
						},
						{
							"id": "aap-app-1",
							"version": "4.3"
						}
					]
				},
				{
					"id": "safety-ecu",
					"components": [
						{
							"id": "powertrain",
							"version": "342.444.195"
						},
						{
							"id": "body",
							"version": "568.484.195"
						}
					]
				},
				{
					"id": "self-update",
					"components": [
						{
							"id": "os-image",
							"version": "1.10-alpha"
						}
					]
				}
			]
		}
	}`

	activityID, desiredState, err := FromDesiredStateBytes([]byte(spec))
	assert.NoError(t, err)

	assert.Equal(t, "random-activity-id", activityID)

	// assert baselines
	assert.Equal(t, 2, len(desiredState.Baselines))
	assert.Equal(t, "powertrain 2022", desiredState.Baselines[0].Title)
	assert.Equal(t, []string{"safety-ecu:powertrain", "safety-ecu:body"}, desiredState.Baselines[0].Components)
	assert.Equal(t, "Composite-app123", desiredState.Baselines[1].Title)
	assert.Equal(t, []string{"containers:xyz", "safety-app:aap-base-1", "safety-app:aap-app-1"}, desiredState.Baselines[1].Components)

	// assert domains
	assert.Equal(t, 4, len(desiredState.Domains))

	// assert containers
	containers := desiredState.Domains[0]
	assert.Equal(t, "containers", containers.ID)
	assert.Equal(t, 1, len(containers.Config))
	assert.Equal(t, "source", containers.Config[0].Key)
	assert.Equal(t, "my-container-registry.com", containers.Config[0].Value)
	assert.Equal(t, 2, len(containers.Components))

	assert.Equal(t, "xyz", containers.Components[0].ID)
	assert.Equal(t, "1", containers.Components[0].Version)
	assert.Equal(t, 2, len(containers.Components[0].Config))
	assert.Equal(t, "key1", containers.Components[0].Config[0].Key)
	assert.Equal(t, "val1", containers.Components[0].Config[0].Value)
	assert.Equal(t, "key2", containers.Components[0].Config[1].Key)
	assert.Equal(t, "val2", containers.Components[0].Config[1].Value)

	assert.Equal(t, "abc", containers.Components[1].ID)
	assert.Equal(t, "4", containers.Components[1].Version)
	assert.Equal(t, 2, len(containers.Components[1].Config))
	assert.Equal(t, "k1", containers.Components[1].Config[0].Key)
	assert.Equal(t, "v1", containers.Components[1].Config[0].Value)
	assert.Equal(t, "k2", containers.Components[1].Config[1].Key)
	assert.Equal(t, "v2", containers.Components[1].Config[1].Value)

	// assert safety-app
	safetyapp := desiredState.Domains[1]
	assert.Equal(t, "safety-app", safetyapp.ID)
	assert.Empty(t, safetyapp.Config)
	assert.Equal(t, 2, len(safetyapp.Components))
	assert.Equal(t, "aap-base-1", safetyapp.Components[0].ID)
	assert.Equal(t, "1.0", safetyapp.Components[0].Version)
	assert.Equal(t, "aap-app-1", safetyapp.Components[1].ID)
	assert.Equal(t, "4.3", safetyapp.Components[1].Version)

	// assert safety-ecu
	safetyecu := desiredState.Domains[2]
	assert.Equal(t, "safety-ecu", safetyecu.ID)
	assert.Empty(t, safetyecu.Config)
	assert.Equal(t, 2, len(safetyecu.Components))
	assert.Equal(t, "powertrain", safetyecu.Components[0].ID)
	assert.Equal(t, "342.444.195", safetyecu.Components[0].Version)
	assert.Equal(t, "body", safetyecu.Components[1].ID)
	assert.Equal(t, "568.484.195", safetyecu.Components[1].Version)

	// assert safety-ecu
	selfupdate := desiredState.Domains[3]
	assert.Equal(t, "self-update", selfupdate.ID)
	assert.Empty(t, selfupdate.Config)
	assert.Equal(t, 1, len(selfupdate.Components))
	assert.Equal(t, "os-image", selfupdate.Components[0].ID)
	assert.Equal(t, "1.10-alpha", selfupdate.Components[0].Version)
}
