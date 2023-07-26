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

package orchestration

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/config"
	"github.com/eclipse-kanto/update-manager/mqtt"
	mocks "github.com/eclipse-kanto/update-manager/test/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	testActivityID = "testActivityId"
)

var defaultInventory = &types.Inventory{
	SoftwareNodes: []*types.SoftwareNode{
		{
			InventoryNode: types.InventoryNode{
				ID:   "device-update-manager",
				Name: "Update Manager",
			},
			Type: "APPLICATION",
		},
		{
			InventoryNode: types.InventoryNode{
				ID:   "",
				Name: "testDomainInventoryName1",
			},
			Type: "APPLICATION",
		},
		{
			InventoryNode: types.InventoryNode{
				ID:   "",
				Name: "testDomainInventoryName2",
			},
			Type: "APPLICATION",
		},
	},
	Associations: []*types.Association{
		{
			SourceID: "device-update-manager",
		},
		{
			SourceID: "device-update-manager",
		},
	},
}

func TestNewUpdateManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cfg := createTestConfig(false, false)
	uaClient := mqtt.NewUpdateAgentClient("device", &mqtt.ConnectionConfig{})
	updateManager := NewUpdateManager("dummyVersion", cfg, uaClient, nil).((*aggregatedUpdateManager))

	assert.Equal(t, "device", updateManager.Name())
	assert.Equal(t, "dummyVersion", updateManager.version)
	assert.Equal(t, cfg, updateManager.cfg)
	assert.NotNil(t, updateManager.domainsInventory)
	assert.NotNil(t, updateManager.rebootManager)
	assert.NotNil(t, updateManager.domainAgents)
	assert.Equal(t, 3, len(updateManager.domainAgents))
}

func TestUpdateManagerName(t *testing.T) {
	cfg := createTestConfig(false, false)

	updateManager := createTestUpdateManager(nil, nil, nil, 0, cfg, nil, nil)
	assert.Equal(t, "device", updateManager.Name())
}

func TestGetCurrentState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	domainUpdateManagers := createTestDomainUpdateManagers(mockCtrl)
	domainInventory := map[string]*types.Inventory{
		"testDomainInventory1": {
			SoftwareNodes: []*types.SoftwareNode{
				{
					InventoryNode: types.InventoryNode{
						Name: "testDomainInventoryName1",
					},
					Type: "APPLICATION",
				},
			},
		},
		"testDomainInventory2": {
			SoftwareNodes: []*types.SoftwareNode{
				{
					InventoryNode: types.InventoryNode{
						Name: "testDomainInventoryName2",
					},
					Type: "APPLICATION",
				},
			},
		},
	}
	for _, domainUpdateManager := range domainUpdateManagers {
		domainUpdateManager.(*mocks.MockUpdateManager).EXPECT().Get(ctx, "test")
		domainUpdateManager.(*mocks.MockUpdateManager).EXPECT().Name().Times(1)
	}

	cfg := createTestConfig(false, false)

	updateManager := createTestUpdateManager(nil, domainUpdateManagers, nil, 0, cfg, nil, domainInventory)
	currentState, err := updateManager.Get(ctx, "test")
	assert.Nil(t, err)
	reflect.DeepEqual(defaultInventory, currentState)
}

func TestApplyDesiredState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	desiredState1 := &types.DesiredState{
		Domains: []*types.Domain{
			{
				ID: "testDomain1",
			},
		},
	}
	testInventory := &types.Inventory{
		SoftwareNodes: []*types.SoftwareNode{
			{
				InventoryNode: types.InventoryNode{
					ID:   "device-update-manager",
					Name: "Update Manager",
				},
				Type: "APPLICATION",
			},
		},
	}
	ctx := context.Background()

	cfg := createTestConfig(false, false)
	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	mockUpdateOrchestrator := mocks.NewMockUpdateOrchestrator(mockCtrl)

	domainUpdateManager := mocks.NewMockUpdateManager(mockCtrl)
	domainUpdateManagers := map[string]api.UpdateManager{"testDomain1": domainUpdateManager}
	domainUpdateManager.EXPECT().Get(ctx, testActivityID).Times(1).Return(nil, nil)
	domainUpdateManager.EXPECT().Name().Return("testDomain1")

	updateManager := createTestUpdateManager(eventCallback, domainUpdateManagers, nil, 0, cfg, mockUpdateOrchestrator, nil)

	mockUpdateOrchestrator.EXPECT().Apply(context.Background(), domainUpdateManagers, testActivityID, desiredState1, eventCallback).Times(1)

	eventCallback.EXPECT().HandleCurrentStateEvent("device", testActivityID, testInventory)

	updateManager.Apply(ctx, testActivityID, desiredState1)
}

func TestDisposeUpdateManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	domainUpdateManagers := createTestDomainUpdateManagers(mockCtrl)

	for _, domainUpdateManager := range domainUpdateManagers {
		domainUpdateManager.(*mocks.MockUpdateManager).EXPECT().Dispose()
	}

	cfg := createTestConfig(false, false)
	updateManager := createTestUpdateManager(nil, domainUpdateManagers, nil, 0, cfg, nil, nil)
	assert.Nil(t, updateManager.Dispose())
}

func TestStartWatchEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	domainUpdateManagers := createTestDomainUpdateManagers(mockCtrl)
	for _, domainUpdateManager := range domainUpdateManagers {
		domainUpdateManager.(*mocks.MockUpdateManager).EXPECT().WatchEvents(ctx)
	}

	cfg := createTestConfig(false, false)
	updateManager := createTestUpdateManager(nil, domainUpdateManagers, nil, 0, cfg, nil, nil)
	updateManager.WatchEvents(ctx)
}

func TestSetCallback(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)

	cfg := createTestConfig(false, false)
	updateManager := createTestUpdateManager(nil, nil, nil, 0, cfg, nil, nil)
	updateManager.SetCallback(eventCallback)
	assert.Equal(t, eventCallback, updateManager.eventCallback)
}

func TestRebootAfterApplyDesiredState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testValues := []struct {
		name           string
		rebootEnabled  bool
		rebootRequired bool
		rebootError    error
		expectedReboot bool
	}{
		{
			"reboot_enabled_and_required_successuful",
			true,
			true,
			nil,
			true,
		},
		{
			"reboot_enabled_and_required_failed",
			true,
			true,
			fmt.Errorf("reboot error"),
			true,
		},
		{
			"reboot_enabled_and_not_required",
			true,
			false,
			nil,
			false,
		},
		{
			"reboot_disabled_and_required",
			false,
			true,
			nil,
			false,
		},
		{
			"reboot_disabled_and_not_required",
			false,
			false,
			nil,
			false,
		},
	}

	for _, testValue := range testValues {
		t.Run(testValue.name, func(t *testing.T) {
			testInventory := &types.Inventory{
				SoftwareNodes: []*types.SoftwareNode{
					{
						InventoryNode: types.InventoryNode{
							ID:   "device-update-manager",
							Name: "Update Manager",
						},
						Type: "APPLICATION",
					},
				},
			}
			eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
			mockUpdateOrchestrator := mocks.NewMockUpdateOrchestrator(mockCtrl)

			eventCallback.EXPECT().HandleCurrentStateEvent("device", testActivityID, testInventory)
			rebootManager := mocks.NewMockRebootManager(mockCtrl)

			if testValue.expectedReboot {
				rebootManager.EXPECT().Reboot(30 * time.Second).Return(testValue.rebootError)
			}
			cfg := createTestConfig(testValue.rebootRequired, testValue.rebootEnabled)
			domainUpdateManagers := map[string]api.UpdateManager{}
			updateManager := createTestUpdateManager(eventCallback, domainUpdateManagers, rebootManager, 0, cfg, mockUpdateOrchestrator, nil)
			mockUpdateOrchestrator.EXPECT().Apply(context.Background(), domainUpdateManagers, testActivityID, nil, eventCallback).Return(testValue.rebootRequired).Times(1)

			updateManager.Apply(context.Background(), testActivityID, nil)
		})
	}
}

func createTestDomainUpdateManagers(mockCtrl *gomock.Controller) map[string]api.UpdateManager {
	domainUpdateManagers := map[string]api.UpdateManager{}
	domainUpdateManager1 := mocks.NewMockUpdateManager(mockCtrl)
	domainUpdateManagers["testDomain1"] = domainUpdateManager1
	domainUpdateManager2 := mocks.NewMockUpdateManager(mockCtrl)
	domainUpdateManagers["testDomain2"] = domainUpdateManager2
	domainUpdateManager3 := mocks.NewMockUpdateManager(mockCtrl)
	domainUpdateManagers["testDomain3"] = domainUpdateManager3
	return domainUpdateManagers
}

func createTestUpdateManager(eventCallback api.UpdateManagerCallback, updateManagers map[string]api.UpdateManager,
	rebootManager RebootManager, reportFeedbackInterval time.Duration, cfg *config.Config, updateOrchestrator api.UpdateOrchestrator, domainsInventory map[string]*types.Inventory) *aggregatedUpdateManager {
	return &aggregatedUpdateManager{
		name:               "device",
		cfg:                cfg,
		rebootManager:      rebootManager,
		domainAgents:       updateManagers,
		eventCallback:      eventCallback,
		updateOrchestrator: updateOrchestrator,
		domainsInventory:   domainsInventory,
	}
}

func createTestConfig(rebootRequired, rebootEnabled bool) *config.Config {
	agents := make(map[string]*api.UpdateManagerConfig)

	agents["testDomain1"] = &api.UpdateManagerConfig{
		Name:           "testDomain1",
		RebootRequired: rebootRequired,
		ReadTimeout:    "1s",
	}
	agents["testDomain2"] = &api.UpdateManagerConfig{
		Name:           "testDomain2",
		RebootRequired: rebootRequired,
		ReadTimeout:    "1s",
	}
	agents["testDomain3"] = &api.UpdateManagerConfig{
		Name:           "testDomain3",
		RebootRequired: rebootRequired,
		ReadTimeout:    "1s",
	}

	return &config.Config{
		BaseConfig: &config.BaseConfig{
			Domain: "device",
		},
		Agents:        agents,
		RebootEnabled: rebootEnabled,
	}
}
