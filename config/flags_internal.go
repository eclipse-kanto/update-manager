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

package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/eclipse-kanto/update-manager/api/types"
	"github.com/eclipse-kanto/update-manager/logger"
)

const (
	// domains flag
	domainsFlagID              = "domains"
	domainsDesc                = "Specify a comma-separated list of domains handled by the update manager"
	ownerConsentCommandsFlagID = "owner-consent-commands"
	ownerConsentCommandsDesc   = "Specify a comma-separated list of commands, before which an owner consent should be granted. Possible values are: 'download', 'update', 'activate'"
)

// SetupAllUpdateManagerFlags adds all flags for the configuration of the update manager
func SetupAllUpdateManagerFlags(flagSet *flag.FlagSet, cfg *Config) {
	SetupFlags(flagSet, cfg.BaseConfig)

	flagSet.String(domainsFlagID, "", domainsDesc)

	flagSet.BoolVar(&cfg.RebootEnabled, "reboot-enabled", EnvToBool("REBOOT_ENABLED", cfg.RebootEnabled), "Specify a flag that controls the enabling/disabling of the reboot process after successful update operation")
	flagSet.StringVar(&cfg.RebootAfter, "reboot-after", EnvToString("REBOOT_AFTER", cfg.RebootAfter), "Specify the timeout in cron format to wait before a reboot process is initiated after successful update operation. Value should be a positive integer number followed by a unit suffix, such as '60s', '10m', etc")

	flagSet.StringVar(&cfg.PhaseTimeout, "phase-timeout", EnvToString("PHASE_TIMEOUT", cfg.PhaseTimeout), "Specify the timeout for completing an Update Orchestration phase. Value should be a positive integer number followed by a unit suffix, such as '60s', '10m', etc")
	flagSet.StringVar(&cfg.ReportFeedbackInterval, "report-feedback-interval", EnvToString("REPORT_FEEDBACK_INTERVAL", cfg.ReportFeedbackInterval), "Specify the time interval for reporting intermediate desired state feedback messages during an active update operation. Value should be a positive integer number followed by a unit suffix, such as '60s', '10m', etc")
	flagSet.StringVar(&cfg.CurrentStateDelay, "current-state-delay", EnvToString("CURRENT_STATE_DELAY", cfg.CurrentStateDelay), "Specify the time delay for reporting current state messages. Value should be a positive integer number followed by a unit suffix, such as '60s', '10m', etc")
	setupAgentsConfigFlags(flagSet, cfg)
}

func parseFlags(cfg *Config, version string) {
	domains := parseDomainsFlag()
	prepareAgentsConfig(cfg, domains)

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagSet := flag.CommandLine

	SetupAllUpdateManagerFlags(flagSet, cfg)

	fVersion := flagSet.Bool("version", false, "Prints current version and exits")
	listCommands := flagSet.String(ownerConsentCommandsFlagID, "", ownerConsentCommandsDesc)
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		logger.ErrorErr(err, "Cannot parse command flags")
	}

	if *fVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if len(*listCommands) != 0 {
		cfg.OwnerConsentCommands = parseOwnerConsentCommandsFlag(*listCommands)
	}
}

func parseOwnerConsentCommandsFlag(listCommands string) []types.CommandType {
	var result []types.CommandType
	for _, command := range strings.Split(listCommands, ",") {
		c := strings.TrimSpace(command)
		if len(c) > 0 {
			result = append(result, types.CommandType(strings.ToUpper(c)))
		}
	}
	return result
}

func parseDomainsFlag() map[string]bool {
	var listDomains string
	flagSet := flag.NewFlagSet("", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	flagSet.StringVar(&listDomains, domainsFlagID, EnvToString("DOMAINS", ""), domainsDesc)
	if err := flagSet.Parse(getFlagArgs(domainsFlagID)); err != nil {
		logger.ErrorErr(err, "Cannot parse domain flag")
	}
	if len(listDomains) > 0 {
		domains := strings.Split(listDomains, ",")
		result := make(map[string]bool)
		for _, domain := range domains {
			domain = strings.TrimSpace(domain)
			if len(domain) > 0 {
				result[domain] = true
			}
		}
		return result
	}
	return nil
}

func getFlagArgs(flag string) []string {
	args := os.Args[1:]
	flag1 := "-" + flag
	flag2 := "--" + flag
	for index, arg := range args {
		if strings.HasPrefix(arg, flag1+"=") || strings.HasPrefix(arg, flag2+"=") {
			return []string{arg}
		}
		if (arg == flag1 || arg == flag2) && index < len(args)-1 {
			return args[index : index+2]
		}
	}
	return []string{}
}

func setupAgentsConfigFlags(flagSet *flag.FlagSet, cfg *Config) {
	for _, agent := range cfg.Agents {
		rr := fmt.Sprintf("%s-reboot-required", agent.Name)
		rto := fmt.Sprintf("%s-read-timeout", agent.Name)
		rrEV := fmt.Sprintf("%s_REBOOT_REQUIRED", strings.ReplaceAll(strings.ToUpper(agent.Name), "-", "_"))
		rtoEV := fmt.Sprintf("%s_READ_TIMEOUT", strings.ReplaceAll(strings.ToUpper(agent.Name), "-", "_"))

		rrDef := agent.RebootRequired
		rtoDef := agent.ReadTimeout
		if rtoDef == "" {
			rtoDef = readTimeoutDefault
		}

		flagSet.BoolVar(&agent.RebootRequired, rr, EnvToBool(rrEV, rrDef), "Specify the reboot required flag for the given domain.")
		flagSet.StringVar(&agent.ReadTimeout, rto, EnvToString(rtoEV, rtoDef), "Specify the read timeout for the given domain. Value should be a positive integer number followed by a unit suffix, such as '60s', '10m', etc")
	}
}
