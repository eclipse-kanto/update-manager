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
	"testing"
	"time"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/config"
	"github.com/eclipse-kanto/update-manager/mqtt"
	"github.com/eclipse-kanto/update-manager/test"
	mocks "github.com/eclipse-kanto/update-manager/test/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var defaultInventory = &types.Inventory{
	SoftwareNodes: []*types.SoftwareNode{
		test.MainInventoryNode,
		test.CreateSoftwareNode("domain", 1, "", "", types.SoftwareTypeApplication),
		test.CreateSoftwareNode("domain", 2, "", "", types.SoftwareTypeApplication),
	},
	Associations: []*types.Association{
		test.CreateAssociation("device-update-manager", "domain-test:1"),
		test.CreateAssociation("device-update-manager", "domain-test:2"),
	},
}

func TestNewUpdateManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cfg := createTestConfig(false, false)
	t.Run("test_no_error", func(t *testing.T) {
		uaClient := mqtt.NewUpdateAgentClient("device", &mqtt.ConnectionConfig{})
		apiUpdateManager, err := NewUpdateManager("dummyVersion", cfg, uaClient, nil)
		assert.NoError(t, err)
		updateManager := apiUpdateManager.(*aggregatedUpdateManager)

		assert.Equal(t, "device", updateManager.Name())
		assert.Equal(t, "dummyVersion", updateManager.version)
		assert.Equal(t, cfg, updateManager.cfg)
		assert.NotNil(t, updateManager.domainsInventory)
		assert.NotNil(t, updateManager.rebootManager)
		assert.NotNil(t, updateManager.domainAgents)
		assert.Equal(t, 3, len(updateManager.domainAgents))
	})
	t.Run("test_error", func(t *testing.T) {
		mockClient := mocks.NewMockUpdateAgentClient(mockCtrl)
		apiUpdateManager, err := NewUpdateManager("dummyVersion", cfg, mockClient, nil)
		assert.Error(t, err)
		assert.Nil(t, apiUpdateManager)
	})
}

func TestGetCurrentState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	domainUpdateManagers := createTestDomainUpdateManagers(mockCtrl)
	domainInventory := map[string]*types.Inventory{
		"testDomainInventory1": {
			SoftwareNodes: []*types.SoftwareNode{
				test.CreateSoftwareNode("domain", 1, "", "", types.SoftwareTypeApplication),
			},
		},
		"testDomainInventory2": {
			SoftwareNodes: []*types.SoftwareNode{
				test.CreateSoftwareNode("domain", 2, "", "", types.SoftwareTypeApplication),
			},
		},
	}
	for _, domainUpdateManager := range domainUpdateManagers {
		domainUpdateManager.(*mocks.MockUpdateManager).EXPECT().Get(ctx, "test")
		domainUpdateManager.(*mocks.MockUpdateManager).EXPECT().Name().Times(1)
	}

	updateManager := createTestUpdateManager(nil, domainUpdateManagers, nil, 0, createTestConfig(false, false), nil, domainInventory, "development")
	currentState, err := updateManager.Get(ctx, "test")
	assert.Nil(t, err)
	test.AssertInventoryWithoutElementsOrder(t, defaultInventory, currentState)
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
			test.MainInventoryNode,
		},
	}
	ctx := context.Background()

	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
	mockUpdateOrchestrator := mocks.NewMockUpdateOrchestrator(mockCtrl)

	domainUpdateManager := mocks.NewMockUpdateManager(mockCtrl)
	domainUpdateManagers := map[string]api.UpdateManager{"testDomain1": domainUpdateManager}
	domainUpdateManager.EXPECT().Get(ctx, test.ActivityID).Times(1).Return(nil, nil)
	domainUpdateManager.EXPECT().Name().Return("testDomain1")

	updateManager := createTestUpdateManager(eventCallback, domainUpdateManagers, nil, 0, createTestConfig(false, false), mockUpdateOrchestrator, nil, "development")

	mockUpdateOrchestrator.EXPECT().Apply(context.Background(), domainUpdateManagers, test.ActivityID, desiredState1, eventCallback).Times(1)

	eventCallback.EXPECT().HandleCurrentStateEvent("device", test.ActivityID, testInventory)

	updateManager.Apply(ctx, test.ActivityID, desiredState1)
}

func TestDisposeUpdateManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	domainUpdateManagers := createTestDomainUpdateManagers(mockCtrl)

	for _, domainUpdateManager := range domainUpdateManagers {
		domainUpdateManager.(*mocks.MockUpdateManager).EXPECT().Dispose()
	}

	updateManager := createTestUpdateManager(nil, domainUpdateManagers, nil, 0, createTestConfig(false, false), nil, nil, "development")
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

	createTestUpdateManager(nil, domainUpdateManagers, nil, 0, createTestConfig(false, false), nil, nil, "1.0.0").WatchEvents(ctx)
}

func TestSetCallback(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)

	updateManager := createTestUpdateManager(nil, nil, nil, 0, createTestConfig(false, false), nil, nil, "1.0.0")
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
			name: "reboot_disabled_and_not_required",
		},
	}

	for _, testValue := range testValues {
		t.Run(testValue.name, func(t *testing.T) {
			testInventory := &types.Inventory{
				SoftwareNodes: []*types.SoftwareNode{
					test.MainInventoryNode,
				},
			}
			eventCallback := mocks.NewMockUpdateManagerCallback(mockCtrl)
			mockUpdateOrchestrator := mocks.NewMockUpdateOrchestrator(mockCtrl)

			eventCallback.EXPECT().HandleCurrentStateEvent("device", test.ActivityID, testInventory)
			rebootManager := mocks.NewMockRebootManager(mockCtrl)

			if testValue.expectedReboot {
				rebootManager.EXPECT().Reboot(30 * time.Second).Return(testValue.rebootError)
			}
			cfg := createTestConfig(testValue.rebootRequired, testValue.rebootEnabled)
			domainUpdateManagers := map[string]api.UpdateManager{}
			updateManager := createTestUpdateManager(eventCallback, domainUpdateManagers, rebootManager, 0, cfg, mockUpdateOrchestrator, nil, "development")
			mockUpdateOrchestrator.EXPECT().Apply(context.Background(), domainUpdateManagers, test.ActivityID, nil, eventCallback).Return(testValue.rebootRequired).Times(1)

			updateManager.Apply(context.Background(), test.ActivityID, nil)
		})
	}
}

func createTestDomainUpdateManagers(mockCtrl *gomock.Controller) map[string]api.UpdateManager {
	domainUpdateManagers := map[string]api.UpdateManager{}
	for i := 1; i < 4; i++ {
		domainUpdateManagers[fmt.Sprintf("testDomain%d", i)] = mocks.NewMockUpdateManager(mockCtrl)
	}
	return domainUpdateManagers
}

func createTestUpdateManager(eventCallback api.UpdateManagerCallback, updateManagers map[string]api.UpdateManager,
	rebootManager RebootManager, reportFeedbackInterval time.Duration, cfg *config.Config, updateOrchestrator api.UpdateOrchestrator, domainsInventory map[string]*types.Inventory, version string) *aggregatedUpdateManager {
	return &aggregatedUpdateManager{
		name:               "device",
		version:            version,
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

	for i := 1; i < 4; i++ {
		agents[fmt.Sprintf("testDomain%d", i)] = &api.UpdateManagerConfig{
			Name:           fmt.Sprintf("testDomain%d", i),
			RebootRequired: rebootRequired,
			ReadTimeout:    test.Interval.String(),
		}
	}

	return &config.Config{
		BaseConfig: &config.BaseConfig{
			Domain: "device",
		},
		Agents:        agents,
		RebootEnabled: rebootEnabled,
	}
}
