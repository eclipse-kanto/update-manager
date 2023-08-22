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
	"sync"

	"github.com/eclipse-kanto/update-manager/api"
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
)

const (
	updateManagerName = "Update Manager"
)

func (updateManager *aggregatedUpdateManager) asSoftwareNode() *types.SoftwareNode {
	return &types.SoftwareNode{
		InventoryNode: types.InventoryNode{
			ID:      updateManager.Name() + "-update-manager",
			Version: updateManager.version,
			Name:    updateManagerName,
		},
		Type: types.SoftwareTypeApplication,
	}
}

func updateInventoryForDomain(ctx context.Context, wg *sync.WaitGroup, activityID string,
	agent api.UpdateManager, domainsInventory map[string]*types.Inventory) {
	defer wg.Done()

	inventory, err := agent.Get(ctx, activityID)
	if err != nil {
		if inventory == nil {
			logger.ErrorErr(err, "cannot get current state for domain %s", agent.Name())
			return
		}
		logger.Warn(err.Error())
	}
	if inventory != nil {
		domainsInventory[agent.Name()] = inventory
		logger.Debug("got current state for domain [%s]", agent.Name())
	} else {
		logger.Warn("got empty current state for domain %s", agent.Name())
	}
}

func toFullInventory(updateManagerNode *types.SoftwareNode, domainsInventory map[string]*types.Inventory) *types.Inventory {
	inventory := &types.Inventory{
		SoftwareNodes: []*types.SoftwareNode{updateManagerNode},
	}
	for domain, domainInventory := range domainsInventory {
		inventory.HardwareNodes = append(inventory.HardwareNodes, domainInventory.HardwareNodes...)
		inventory.SoftwareNodes = append(inventory.SoftwareNodes, domainInventory.SoftwareNodes...)
		rootNode := findDomainSoftwareNode(domain, domainInventory)
		if rootNode == nil {
			continue
		}
		inventory.Associations = append(inventory.Associations, &types.Association{SourceID: updateManagerNode.ID, TargetID: rootNode.ID})
		inventory.Associations = append(inventory.Associations, domainInventory.Associations...)
	}
	return inventory
}

func findDomainSoftwareNode(domain string, inventory *types.Inventory) *types.SoftwareNode {
	if len(inventory.SoftwareNodes) == 0 {
		return nil
	}
	for _, softwareNode := range inventory.SoftwareNodes {
		if softwareNode.Type != types.SoftwareTypeApplication {
			continue
		}
		for _, parameter := range softwareNode.Parameters {
			if parameter.Key != "domain" {
				continue
			}
			if parameter.Value != domain {
				continue
			}
			return softwareNode
		}
	}
	return inventory.SoftwareNodes[0]
}
