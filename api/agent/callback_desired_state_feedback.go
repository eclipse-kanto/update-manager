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

package agent

import (
	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
)

func (agent *updateAgent) HandleDesiredStateFeedbackEvent(domain string, activityID string, baseline string, status types.StatusType, message string, actions []*types.Action) {
	logger.Debug("handle desired state feedback event for domain and activityId '%s' - '%s'", domain, activityID)

	if status != types.StatusRunning {
		agent.publishDesiredStateFeedback(activityID, &types.DesiredStateFeedback{
			Baseline: baseline,
			Status:   status,
			Message:  message,
			Actions:  actions,
		})
		if status == types.StatusCompleted || status == types.StatusIncomplete {
			if agent.desiredStateFeedbackNotifier != nil {
				agent.desiredStateFeedbackNotifier.stop()
			}
		}
		return
	}

	if agent.desiredStateFeedbackReportInterval <= 0 {
		return
	}

	if agent.desiredStateFeedbackNotifier == nil {
		agent.desiredStateFeedbackNotifier = newDesiredStateFeedbackNotifier(agent.desiredStateFeedbackReportInterval, agent)
	}
	agent.desiredStateFeedbackNotifier.set(activityID, actions)
}

func (agent *updateAgent) publishDesiredStateFeedback(activityID string, feedback *types.DesiredStateFeedback) {
	desiredStateFeedbackBytes, err := types.ToDesiredStateFeedbackBytes(activityID, feedback)
	if err != nil {
		logger.ErrorErr(err, "cannot create payload for desired state feedback.")
		return
	}
	if err := agent.client.PublishDesiredStateFeedback(desiredStateFeedbackBytes); err != nil {
		logger.ErrorErr(err, "cannot publish desired state feedback.")
	}
}
