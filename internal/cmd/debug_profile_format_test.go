// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Tests for --profile and --profile-format validation in the debug command's
// PreRunE, and for trace export wiring (--generate-trace / --trace-output).

package cmd

import (
	"strings"
	"testing"
)

// cleanDebugProfileFlags resets the flags touched by profile/trace-export tests.
func cleanDebugProfileFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		ProfileFlag = false
		ProfileFormatFlag = "html"
		generateTrace = false
		traceOutputFile = ""
		networkFlag = "mainnet"
		wasmPath = ""
		demoMode = false
		xdrFileFlag = ""
		jsonFileFlag = ""
		loadSnapshotsFlag = ""
		hotReloadFlag = false
		opIndexFlag = -1
		compareNetworkFlag = ""
		watchFlag = false
		watchTimeoutFlag = 30
		traceVerbosityFlag = "normal"
		debugFormatFlag = "text"
	})
}

// ── --profile-format validation ───────────────────────────────────────────────

// TestDebugPreRunE_ProfileFormat_HTML_Valid verifies html is accepted.
func TestDebugPreRunE_ProfileFormat_HTML_Valid(t *testing.T) {
	cleanDebugProfileFlags(t)
	ProfileFlag = true
	ProfileFormatFlag = "html"
	networkFlag = "testnet"

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err != nil && strings.Contains(err.Error(), "profile-format") {
		t.Errorf("--profile-format=html should be accepted, got: %v", err)
	}
}

// TestDebugPreRunE_ProfileFormat_SVG_Valid verifies svg is accepted.
func TestDebugPreRunE_ProfileFormat_SVG_Valid(t *testing.T) {
	cleanDebugProfileFlags(t)
	ProfileFlag = true
	ProfileFormatFlag = "svg"
	networkFlag = "testnet"

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err != nil && strings.Contains(err.Error(), "profile-format") {
		t.Errorf("--profile-format=svg should be accepted, got: %v", err)
	}
}

// TestDebugPreRunE_ProfileFormat_CaseInsensitive verifies HTML / SVG (upper) are accepted.
func TestDebugPreRunE_ProfileFormat_CaseInsensitive(t *testing.T) {
	for _, fmt := range []string{"HTML", "SVG", "Html", "Svg"} {
		fmt := fmt
		t.Run(fmt, func(t *testing.T) {
			cleanDebugProfileFlags(t)
			ProfileFlag = true
			ProfileFormatFlag = fmt
			networkFlag = "testnet"

			validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
			err := debugCmd.PreRunE(debugCmd, []string{validHash})
			if err != nil && strings.Contains(err.Error(), "profile-format") {
				t.Errorf("--profile-format=%s should be accepted (case-insensitive), got: %v", fmt, err)
			}
		})
	}
}

// TestDebugPreRunE_ProfileFormat_Invalid_PDF verifies an unknown format is rejected.
func TestDebugPreRunE_ProfileFormat_Invalid_PDF(t *testing.T) {
	cleanDebugProfileFlags(t)
	ProfileFlag = true
	ProfileFormatFlag = "pdf"
	networkFlag = "testnet"

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err == nil {
		t.Fatal("expected error for invalid --profile-format=pdf")
	}
	msg := err.Error()
	if !strings.Contains(msg, "profile-format") {
		t.Errorf("error should mention 'profile-format', got: %q", msg)
	}
	if !strings.Contains(msg, "pdf") {
		t.Errorf("error should echo the invalid value 'pdf', got: %q", msg)
	}
	if !strings.Contains(msg, "html") || !strings.Contains(msg, "svg") {
		t.Errorf("error should list valid values (html, svg), got: %q", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %q", msg)
	}
}

// TestDebugPreRunE_ProfileFormat_Invalid_JSON verifies json is not an accepted
// profile format (it's for --format, not --profile-format).
func TestDebugPreRunE_ProfileFormat_Invalid_JSON(t *testing.T) {
	cleanDebugProfileFlags(t)
	ProfileFlag = true
	ProfileFormatFlag = "json"
	networkFlag = "testnet"

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err == nil {
		t.Fatal("expected error for --profile-format=json (invalid for flamegraph)")
	}
	if !strings.Contains(err.Error(), "profile-format") {
		t.Errorf("error should mention 'profile-format', got: %v", err)
	}
}

// TestDebugPreRunE_ProfileFormat_Empty_WhenProfileEnabled verifies an empty
// --profile-format is treated as valid (defaults to html at runtime).
func TestDebugPreRunE_ProfileFormat_Empty_WhenProfileEnabled(t *testing.T) {
	cleanDebugProfileFlags(t)
	ProfileFlag = true
	ProfileFormatFlag = "" // empty → allowed, defaults to html
	networkFlag = "testnet"

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err != nil && strings.Contains(err.Error(), "profile-format") {
		t.Errorf("empty --profile-format should not error when --profile is set, got: %v", err)
	}
}

// TestDebugPreRunE_ProfileFormat_NotValidated_WhenProfileDisabled verifies
// that --profile-format is not validated when --profile is false (flag is
// irrelevant without --profile).
func TestDebugPreRunE_ProfileFormat_NotValidated_WhenProfileDisabled(t *testing.T) {
	cleanDebugProfileFlags(t)
	ProfileFlag = false        // --profile not set
	ProfileFormatFlag = "bad"  // would be invalid IF --profile were set
	networkFlag = "testnet"

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	// Should NOT return a profile-format error because --profile is off.
	if err != nil && strings.Contains(err.Error(), "profile-format") {
		t.Errorf("profile-format should not be validated when --profile=false, got: %v", err)
	}
}

// ── --trace-output path validation ───────────────────────────────────────────

// TestDebugPreRunE_TraceOutput_DirectoryPath_Rejected verifies that a path
// ending in "/" is rejected with a clear error.
func TestDebugPreRunE_TraceOutput_DirectoryPath_Rejected(t *testing.T) {
	cleanDebugProfileFlags(t)
	networkFlag = "testnet"
	traceOutputFile = "./traces/" // directory path — should be rejected

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err == nil {
		t.Fatal("expected error for --trace-output that looks like a directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention 'directory', got: %v", err)
	}
}

// TestDebugPreRunE_TraceOutput_FilePath_Accepted verifies a valid file path
// passes validation.
func TestDebugPreRunE_TraceOutput_FilePath_Accepted(t *testing.T) {
	cleanDebugProfileFlags(t)
	networkFlag = "testnet"
	traceOutputFile = "./output/trace.json"

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err != nil && strings.Contains(err.Error(), "trace-output") {
		t.Errorf("valid --trace-output path should not fail, got: %v", err)
	}
}

// TestDebugPreRunE_TraceOutput_PathTraversal_Rejected verifies that a
// path traversal sequence in --trace-output is caught.
func TestDebugPreRunE_TraceOutput_PathTraversal_Rejected(t *testing.T) {
	cleanDebugProfileFlags(t)
	networkFlag = "testnet"
	traceOutputFile = "../../../etc/sensitive.json"

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err == nil {
		t.Fatal("expected error for --trace-output with path traversal")
	}
	if !strings.Contains(err.Error(), "..") {
		t.Errorf("error should mention '..' traversal sequence, got: %v", err)
	}
}
