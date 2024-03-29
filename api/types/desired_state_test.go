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
		"timestamp": 16656700000000,
		"payload": {
			"baselines": [
				{
					"title": "simple-baseline",
					"components": [
						"test-domain:component1",
						"test-domain:component2"
					]
				},
				{
					"title": "composite-baseline",
					"components": [
						"containers:xyz",
						"another-domain:app1",
						"another-domain:app2"
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
					"id": "another-domain",
					"components": [
						{
							"id": "app1",
							"version": "1.0"
						},
						{
							"id": "app2",
							"version": "4.3"
						}
					]
				},
				{
					"id": "test-domain",
					"components": [
						{
							"id": "component1",
							"version": "342.444.195"
						},
						{
							"id": "component2",
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

	desiredState := &DesiredState{}
	envelope, err := FromEnvelope([]byte(spec), desiredState)
	assert.NoError(t, err)

	assert.Equal(t, "random-activity-id", envelope.ActivityID)

	// assert baselines
	assert.Equal(t, 2, len(desiredState.Baselines))
	assert.Equal(t, "simple-baseline", desiredState.Baselines[0].Title)
	assert.Equal(t, []string{"test-domain:component1", "test-domain:component2"}, desiredState.Baselines[0].Components)
	assert.Equal(t, "composite-baseline", desiredState.Baselines[1].Title)
	assert.Equal(t, []string{"containers:xyz", "another-domain:app1", "another-domain:app2"}, desiredState.Baselines[1].Components)

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
	assert.Equal(t, "another-domain", safetyapp.ID)
	assert.Empty(t, safetyapp.Config)
	assert.Equal(t, 2, len(safetyapp.Components))
	assert.Equal(t, "app1", safetyapp.Components[0].ID)
	assert.Equal(t, "1.0", safetyapp.Components[0].Version)
	assert.Equal(t, "app2", safetyapp.Components[1].ID)
	assert.Equal(t, "4.3", safetyapp.Components[1].Version)

	// assert domain1
	safetyecu := desiredState.Domains[2]
	assert.Equal(t, "test-domain", safetyecu.ID)
	assert.Empty(t, safetyecu.Config)
	assert.Equal(t, 2, len(safetyecu.Components))
	assert.Equal(t, "component1", safetyecu.Components[0].ID)
	assert.Equal(t, "342.444.195", safetyecu.Components[0].Version)
	assert.Equal(t, "component2", safetyecu.Components[1].ID)
	assert.Equal(t, "568.484.195", safetyecu.Components[1].Version)

	// assert domain1
	selfupdate := desiredState.Domains[3]
	assert.Equal(t, "self-update", selfupdate.ID)
	assert.Empty(t, selfupdate.Config)
	assert.Equal(t, 1, len(selfupdate.Components))
	assert.Equal(t, "os-image", selfupdate.Components[0].ID)
	assert.Equal(t, "1.10-alpha", selfupdate.Components[0].Version)
}

func TestSplitPerDomains(t *testing.T) {
	desiredState := &DesiredState{
		Domains: []*Domain{
			{
				ID: "TestDomain1",
			},
			{
				ID: "TestDomain2",
			},
		},
		Baselines: []*Baseline{
			{
				Title:         "TestBaseline 1",
				Description:   "TestDescription 1",
				Preconditions: "TestPrecondition 1",
				Components: []string{
					"TestDomain1:Component1",
					"TestDomain1:Component2",
				},
			},
			{
				Title:         "TestBaseline 3",
				Description:   "TestDescription 3",
				Preconditions: "TestPrecondition 3",
				Components: []string{
					"TestDomain1:Component3",
					"TestDomain1:Component3",
				},
			},
			{
				Title:         "TestBaseline 2",
				Description:   "TestDescription 2",
				Preconditions: "TestPrecondition 2",
				Components: []string{
					"TestDomain2:Component3",
				},
			},
		},
	}
	t.Run("test_split_per_domains", func(t *testing.T) {
		expected := map[string]*DesiredState{}
		expected["TestDomain1"] = &DesiredState{
			Domains: []*Domain{
				{
					ID: "TestDomain1",
				},
			},

			Baselines: []*Baseline{
				{
					Title:         "TestBaseline 1",
					Description:   "TestDescription 1",
					Preconditions: "TestPrecondition 1",
					Components: []string{
						"TestDomain1:Component1",
						"TestDomain1:Component2",
					},
				},
				{
					Title:         "TestBaseline 3",
					Description:   "TestDescription 3",
					Preconditions: "TestPrecondition 3",
					Components: []string{
						"TestDomain1:Component3",
						"TestDomain1:Component3",
					},
				},
			},
		}
		expected["TestDomain2"] = &DesiredState{
			Domains: []*Domain{
				{
					ID: "TestDomain2",
				},
			},

			Baselines: []*Baseline{
				{
					Title:         "TestBaseline 2",
					Description:   "TestDescription 2",
					Preconditions: "TestPrecondition 2",
					Components: []string{
						"TestDomain2:Component3",
					},
				},
			},
		}
		result := desiredState.SplitPerDomains()
		assert.Equal(t, len(expected), len(result))
		for i, ds := range result {
			assert.Equal(t, expected[i].Domains, ds.Domains)
			assert.Equal(t, expected[i].Baselines, ds.Baselines)
		}
	})
	t.Run("test_split_per_domains_nil_domains", func(t *testing.T) {
		desiredState.Domains = nil
		assert.Equal(t, map[string]*DesiredState{}, desiredState.SplitPerDomains())

	})
}

func TestGetBaselinesForDomain(t *testing.T) {
	desiredState := &DesiredState{
		Baselines: []*Baseline{
			{
				Title:         "TestBaseline 1",
				Description:   "TestDescription 1",
				Preconditions: "TestPrecondition 1",
				Components: []string{
					"TestDomain1:Component1",
					"TestDomain1:Component2",
				},
			},
			{
				Title:         "TestBaseline 2",
				Description:   "TestDescription 2",
				Preconditions: "TestPrecondition 2",
				Components: []string{
					"TestDomain2:Component3",
				},
			},
		},
	}

	expected := []*Baseline{
		{
			Title:         "TestBaseline 1",
			Description:   "TestDescription 1",
			Preconditions: "TestPrecondition 1",
			Components: []string{
				"TestDomain1:Component1",
				"TestDomain1:Component2",
			},
		},
	}

	t.Run("test_get_baselines_for_domain_no_err", func(t *testing.T) {
		result := desiredState.GetBaselinesForDomain("TestDomain1")
		assert.Equal(t, len(expected), len(result))
		for i, baseline := range result {
			assert.Equal(t, expected[i].Title, baseline.Title)
			assert.Equal(t, expected[i].Description, baseline.Description)
			assert.Equal(t, expected[i].Preconditions, baseline.Preconditions)
			assert.Equal(t, expected[i].Components, baseline.Components)
		}
	})

	t.Run("test_get_baselines_for_domain_invalid_domain_err", func(t *testing.T) {
		result := desiredState.GetBaselinesForDomain("InvalidDomain")
		assert.Nil(t, result, "Result should be nil")
	})
}
