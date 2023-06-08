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
	"syscall"
	"time"

	"github.com/eclipse-kanto/update-manager/config"
	"github.com/eclipse-kanto/update-manager/logger"

	"github.com/pkg/errors"
)

const (
	defaultSuffixSysRq        = "/sys/kernel/sysrq"
	defaultSuffixSysRqTrigger = "/sysrq-trigger"
)

// RebootManager defines an interface for restarting the host system
type RebootManager interface {
	Reboot(time.Duration) error
}

type rebootManager struct{}

// Reboot makes the host device to reboot.
// Implementation uses the Linux Magic SysRq key combination:
// First, it writes 1 to kernel sysrq file (/proc/sys/kernel/sysrq) to enable function.
// Second, it writes b to sysrg-trigger file (/proc/sysrq-trigger) to reboot the system.
// If the update manager runs inside a container, the given system files shall be mounted from the host to the container.
// It is possible to configure the paths to both files using ENVs FILE_SYS_RQ and FILE_SYS_RQ_TRIGGER.
// If any of these ENVs is set to empty string, then syscall with be tries to perfoem the reboot.
func (rebootManager *rebootManager) Reboot(timeout time.Duration) error {
	logger.Debug("the system is about to reboot after successful update operation in '%s'", timeout)
	<-time.After(timeout)

	fileSysRq, fileSysRqTrigger := getSysRqFiles()
	if fileSysRq == "" || fileSysRqTrigger == "" {
		syscall.Sync()
		return syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	}
	if err := os.WriteFile(fileSysRq, []byte("1"), 0644); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot reboot after successful update operation. cannot send signal to %v.", fileSysRq))
	}
	if err := os.WriteFile(fileSysRqTrigger, []byte("b"), 0200); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot reboot after successful update operation. cannot send signal to %v.", fileSysRqTrigger))
	}
	return nil
}

func getSysRqFiles() (string, string) {
	proc := "/proc"
	if _, err := os.Stat(proc); os.IsNotExist(err) {
		// if update manager runs as a container, usually it may not be allowed to write directly to host's /proc from container.
		// using mount point /proc:wproc solves such restrictions.
		proc = "/wproc"
	}
	return config.EnvToString("FILE_SYS_RQ", proc+defaultSuffixSysRq), config.EnvToString("FILE_SYS_RQ_TRIGGER", proc+defaultSuffixSysRqTrigger)
}
