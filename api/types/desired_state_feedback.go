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

// StatusType defines values for status within the desired state feedback
type StatusType string

// ActionStatusType defines values for status of an action within the desired state feedback
type ActionStatusType string

const (
	// StatusIdentifying denotes that action identification process has started.
	StatusIdentifying StatusType = "IDENTIFYING"
	// StatusIdentified denotes that required actions are identified.
	StatusIdentified StatusType = "IDENTIFIED"
	// StatusIdentificationFailed denotes some or all of the required actions failed to be identified.
	StatusIdentificationFailed StatusType = "IDENTIFICATION_FAILED"
	// StatusDownloadCompleted denotes that identified actions are downloaded successfully.
	StatusDownloadCompleted StatusType = "DOWNLOAD_COMPLETED"
	// StatusDownloadFailed denotes that not all of the identified actions are downloaded successfully.
	StatusDownloadFailed StatusType = "DOWNLOAD_FAILED"
	// StatusRunning denotes that identified actions are currently executing.
	StatusRunning StatusType = "RUNNING"
	// StatusUpdateCompleted denotes that identified actions are updated successfully.
	StatusUpdateCompleted StatusType = "UPDATE_COMPLETED"
	// StatusUpdateFailed denotes that not all identified actions are updated successfully.
	StatusUpdateFailed StatusType = "UPDATE_FAILED"
	// StatusActivationCompleted denotes that identified actions are activated successfully.
	StatusActivationCompleted StatusType = "ACTIVATION_COMPLETED"
	// StatusActivationFailed denotes that not all identified actions are activated successfully.
	StatusActivationFailed StatusType = "ACTIVATION_FAILED"
	// StatusCompleted denotes that identified actions completed successfully.
	StatusCompleted StatusType = "COMPLETED"
	// StatusIncomplete denotes that not all of the identified actions completed successfully.
	StatusIncomplete StatusType = "INCOMPLETE"
	// StatusIncompleteInconsistent denotes that not all of the identified actions completed successfully, leaving the state inconsistent.
	StatusIncompleteInconsistent StatusType = "INCOMPLETE_INCONSISTENT"
	// StatusSuperseded denotes that the identified actions are no longer valid because new desired state was requested.
	StatusSuperseded StatusType = "SUPERSEDED"

	// BaselineStatusDownloading denotes a baseline is currently being downloaded.
	BaselineStatusDownloading StatusType = "DOWNLOADING"
	// BaselineStatusDownloadSuccess denotes a baseline was downloaded successfully.
	BaselineStatusDownloadSuccess StatusType = "DOWNLOAD_SUCCESS"
	// BaselineStatusDownloadFailure denotes a baseline download has failed.
	BaselineStatusDownloadFailure StatusType = "DOWNLOAD_FAILURE"
	// BaselineStatusRollback denotes that a baseline is currently in a rollback process.
	BaselineStatusRollback StatusType = "ROLLBACK"
	// BaselineStatusRollbackSuccess denotes that the rollback process for a baseline was successful.
	BaselineStatusRollbackSuccess StatusType = "ROLLBACK_SUCCESS"
	// BaselineStatusRollbackFailure denotes that the rollback process for a baseline was not successful.
	BaselineStatusRollbackFailure StatusType = "ROLLBACK_FAILURE"
	// BaselineStatusUpdating denotes a baseline is currently being updated.
	BaselineStatusUpdating StatusType = "UPDATING"
	// BaselineStatusUpdateSuccess denotes a baseline has been updated successfully.
	BaselineStatusUpdateSuccess StatusType = "UPDATE_SUCCESS"
	// BaselineStatusUpdateFailure denotes a baseline could not be updated.
	BaselineStatusUpdateFailure StatusType = "UPDATE_FAILURE"
	// BaselineStatusActivating denotes a baseline is currently being activated.
	BaselineStatusActivating StatusType = "ACTIVATING"
	// BaselineStatusActivationSuccess denotes a baseline has been activated successfully.
	BaselineStatusActivationSuccess StatusType = "ACTIVATION_SUCCESS"
	// BaselineStatusActivationFailure denotes a baseline could not be activated.
	BaselineStatusActivationFailure StatusType = "ACTIVATION_FAILURE"
	// BaselineStatusCleanup denotes that the cleanup process for a baseline after activation is currently running.
	BaselineStatusCleanup StatusType = "CLEANUP"
	// BaselineStatusCleanupSuccess denotes that the cleanup process for a baseline after activation was successful.
	BaselineStatusCleanupSuccess StatusType = "CLEANUP_SUCCESS"
	// BaselineStatusCleanupFailure denotes that the cleanup process for a baseline after activation was not successful.
	BaselineStatusCleanupFailure StatusType = "CLEANUP_FAILURE"

	// ActionStatusIdentified denotes action is identified, initial status for each action.
	ActionStatusIdentified ActionStatusType = "IDENTIFIED"
	// ActionStatusDownloading denotes an artifact is currently being downloaded.
	ActionStatusDownloading ActionStatusType = "DOWNLOADING"
	// ActionStatusDownloadFailure denotes an artifact download has failed.
	ActionStatusDownloadFailure ActionStatusType = "DOWNLOAD_FAILURE"
	// ActionStatusDownloadSuccess denotes an artifact has been downloaded successfully.
	ActionStatusDownloadSuccess ActionStatusType = "DOWNLOAD_SUCCESS"
	// ActionStatusUpdating denotes a component is currently being installed or modified.
	ActionStatusUpdating ActionStatusType = "UPDATING"
	// ActionStatusUpdateFailure denotes a component could not be installed or modified.
	ActionStatusUpdateFailure ActionStatusType = "UPDATE_FAILURE"
	// ActionStatusUpdateSuccess denotes a component has been installed or modified successfully.
	ActionStatusUpdateSuccess ActionStatusType = "UPDATE_SUCCESS"
	// ActionStatusRemoving denotes a component is currently being removed.
	ActionStatusRemoving ActionStatusType = "REMOVING"
	// ActionStatusRemovalFailure denotes a component could not be removed.
	ActionStatusRemovalFailure ActionStatusType = "REMOVAL_FAILURE"
	// ActionStatusRemovalSuccess denotes a component has been removed successfully.
	ActionStatusRemovalSuccess ActionStatusType = "REMOVAL_SUCCESS"
	// ActionStatusActivating denotes a component is currently being activated.
	ActionStatusActivating ActionStatusType = "ACTIVATING"
	// ActionStatusActivationFailure denotes a component could not be activated.
	ActionStatusActivationFailure ActionStatusType = "ACTIVATION_FAILURE"
	// ActionStatusActivationSuccess denotes a component has been activated successfully.
	ActionStatusActivationSuccess ActionStatusType = "ACTIVATION_SUCCESS"
)

// DesiredStateFeedback defines the payload holding Desired State Feedback responses.
type DesiredStateFeedback struct {
	Baseline string     `json:"baseline,omitempty"`
	Status   StatusType `json:"status,omitempty"`
	Message  string     `json:"message,omitempty"`
	Actions  []*Action  `json:"actions,omitempty"`
}

// Action defines the payload holding Desired State Feedback responses.
type Action struct {
	Component *Component       `json:"component,omitempty"`
	Status    ActionStatusType `json:"status,omitempty"`
	Progress  uint8            `json:"progress,omitempty"`
	Message   string           `json:"message,omitempty"`
}
