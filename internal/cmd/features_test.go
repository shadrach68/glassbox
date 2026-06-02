// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtocolVerifyCmd_Alias(t *testing.T) {
	assert.Contains(t, protocolVerifyCmd.Aliases, "verify-protocol-registration")
}

func TestProtocolVerifyCmd_ProbeFlagRegistered(t *testing.T) {
	flag := protocolVerifyCmd.Flags().Lookup("probe")
	require.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
}

func TestRunDebugDryRun_InvalidHash(t *testing.T) {
	prevNetwork := networkFlag
	prevCompare := compareNetworkFlag
	t.Cleanup(func() {
		networkFlag = prevNetwork
		compareNetworkFlag = prevCompare
	})

	networkFlag = "testnet"
	compareNetworkFlag = ""

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := runDebugDryRun(cmd, "not-a-valid-hash")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dry-run validation failed")
}

func TestRunDebugDryRun_InvalidNetwork(t *testing.T) {
	prevNetwork := networkFlag
	t.Cleanup(func() { networkFlag = prevNetwork })

	networkFlag = "invalid-network"
	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := runDebugDryRun(cmd, validHash)
	assert.Error(t, err)
}

func TestDebugDryRunFlagRegistered(t *testing.T) {
	flag := debugCmd.Flags().Lookup("dry-run")
	require.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
}

func TestTraceExportMarkdownFlagRegistered(t *testing.T) {
	flag := traceCmd.Flags().Lookup("export-markdown")
	require.NotNil(t, flag)
	assert.Equal(t, "", flag.DefValue)
}

func TestTraceExportFormatFlagRegistered(t *testing.T) {
	flag := traceCmd.Flags().Lookup("export-format")
	require.NotNil(t, flag)
	assert.Equal(t, "html", flag.DefValue)
}
