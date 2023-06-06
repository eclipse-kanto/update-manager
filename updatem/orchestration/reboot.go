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
	"fmt"
	"os"
	"time"

	"github.com/eclipse-kanto/update-manager/logger"

	"github.com/pkg/errors"
)

const (
	procSysRqFile        = "/wproc/sys/kernel/sysrq"
	procSysRqTriggerFile = "/wproc/sysrq-trigger"
)

// RebootManager defines an interface for restarting the host system
type RebootManager interface {
	Reboot(time.Duration) error
}

type rebootManager struct{}

func (rebootManager *rebootManager) Reboot(timeout time.Duration) error {
	logger.Debug("the system is about to reboot after successful update operation in '%s'", timeout)
	<-time.After(timeout)
	if err := os.WriteFile(procSysRqFile, []byte("1"), 0644); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot reboot after successful update operation. cannot send signal to %v.", procSysRqFile))
	}
	if err := os.WriteFile(procSysRqTriggerFile, []byte("b"), 0200); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot reboot after successful update operation. cannot send signal to %v.", procSysRqTriggerFile))
	}
	return nil
}
