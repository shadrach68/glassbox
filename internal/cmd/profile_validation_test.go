// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Tests for Part B (CLI ergonomics) and Part C (performance & profiling) —
// profile command PreRunE validation.

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetProfileFlags restores profile command package-level flag variables to
// their defaults so each test starts from a clean state.
func resetProfileFlags() {
	profileTraceFile = ""
	profileOutput = "profile.pb.gz"
	profileXdr = ""
	profileNetwork = "mainnet"
	profileRPCURL = ""
	profileRPCToken = ""
	profileOutJSON = ""
}

// ── No-input guard ────────────────────────────────────────────────────────────

// TestProfilePreRunE_NoInput verifies that running profile with no flags and
// no positional args returns a clear, actionable error.
func TestProfilePreRunE_NoInput(t *testing.T) {
	t.Cleanup(resetProfileFlags)

	err := profileCmd.PreRunE(profileCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no input is provided")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--xdr") {
		t.Errorf("error should mention --xdr option, got: %q", msg)
	}
	if !strings.Contains(msg, "--file") {
		t.Errorf("error should mention --file option, got: %q", msg)
	}
	// Should include a usage pointer.
	if !strings.Contains(msg, "glassbox profile") {
		t.Errorf("error should include usage hint, got: %q", msg)
	}
}

// ── Mutual exclusion guard ────────────────────────────────────────────────────

// TestProfilePreRunE_XDRAndFile_MutuallyExclusive verifies that providing both
// --xdr and --file is rejected with a clear message.
func TestProfilePreRunE_XDRAndFile_MutuallyExclusive(t *testing.T) {
	t.Cleanup(resetProfileFlags)
	profileXdr = "some-base64-xdr"
	profileTraceFile = "trace.json"

	err := profileCmd.PreRunE(profileCmd, []string{})
	if err == nil {
		t.Fatal("expected error when both --xdr and --file are provided")
	}
	msg := err.Error()
	if !strings.Contains(msg, "mutually exclusive") {
		t.Errorf("error should say mutually exclusive, got: %q", msg)
	}
}

// TestProfilePreRunE_XDRAndPositionalArg_MutuallyExclusive verifies that
// providing --xdr together with a positional trace file is also rejected.
func TestProfilePreRunE_XDRAndPositionalArg_MutuallyExclusive(t *testing.T) {
	t.Cleanup(resetProfileFlags)
	profileXdr = "some-base64-xdr"

	err := profileCmd.PreRunE(profileCmd, []string{"trace.json"})
	if err == nil {
		t.Fatal("expected error when --xdr and a positional file are both given")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error should say mutually exclusive, got: %q", err.Error())
	}
}

// ── Network validation ────────────────────────────────────────────────────────

// TestProfilePreRunE_XDR_InvalidNetwork verifies that an invalid --network
// value is caught before any simulation starts.
func TestProfilePreRunE_XDR_InvalidNetwork(t *testing.T) {
	t.Cleanup(resetProfileFlags)
	profileXdr = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==" // non-empty placeholder
	profileNetwork = "devnet"

	err := profileCmd.PreRunE(profileCmd, []string{})
	if err == nil {
		t.Fatal("expected error for invalid --network")
	}
	msg := err.Error()
	if !strings.Contains(msg, "devnet") {
		t.Errorf("error should include the invalid value, got: %q", msg)
	}
	if !strings.Contains(msg, "testnet") {
		t.Errorf("error should list valid networks, got: %q", msg)
	}
}

// TestProfilePreRunE_XDR_ValidNetworks verifies each valid network passes.
func TestProfilePreRunE_XDR_ValidNetworks(t *testing.T) {
	for _, net := range []string{"testnet", "mainnet", "futurenet"} {
		t.Run(net, func(t *testing.T) {
			t.Cleanup(resetProfileFlags)
			profileXdr = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==" // non-empty placeholder
			profileNetwork = net

			err := profileCmd.PreRunE(profileCmd, []string{})
			// Network itself is valid — may still fail on XDR content but NOT on network name.
			if err != nil && strings.Contains(err.Error(), "invalid --network") {
				t.Errorf("valid network %q should not produce a network error, got: %v", net, err)
			}
		})
	}
}

// ── Missing trace-file guard ──────────────────────────────────────────────────

// TestProfilePreRunE_MissingTraceFile verifies a non-existent trace file path
// produces a clear error with a remediation tip.
func TestProfilePreRunE_MissingTraceFile(t *testing.T) {
	t.Cleanup(resetProfileFlags)
	profileTraceFile = "/nonexistent/path/trace.json"

	err := profileCmd.PreRunE(profileCmd, []string{})
	if err == nil {
		t.Fatal("expected error for non-existent trace file")
	}
	msg := err.Error()
	if !strings.Contains(msg, "/nonexistent/path/trace.json") {
		t.Errorf("error should name the missing file, got: %q", msg)
	}
	// Should include a helpful tip about how to generate a trace.
	if !strings.Contains(msg, "glassbox debug") {
		t.Errorf("error should suggest how to generate a trace, got: %q", msg)
	}
}

// TestProfilePreRunE_MissingPositionalFile verifies a non-existent positional
// trace file argument produces a clear error.
func TestProfilePreRunE_MissingPositionalFile(t *testing.T) {
	t.Cleanup(resetProfileFlags)

	err := profileCmd.PreRunE(profileCmd, []string{"/no/such/file.json"})
	if err == nil {
		t.Fatal("expected error for non-existent positional file")
	}
	if !strings.Contains(err.Error(), "/no/such/file.json") {
		t.Errorf("error should name the missing file, got: %q", err.Error())
	}
}

// TestProfilePreRunE_ExistingTraceFile_Passes verifies a real, readable file
// passes the PreRunE trace-file check.
func TestProfilePreRunE_ExistingTraceFile_Passes(t *testing.T) {
	t.Cleanup(resetProfileFlags)

	tmp := filepath.Join(t.TempDir(), "trace.json")
	if err := os.WriteFile(tmp, []byte(`{"states":[]}`), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	profileTraceFile = tmp

	err := profileCmd.PreRunE(profileCmd, []string{})
	// PreRunE should pass — the file is readable.
	// (RunE will fail on parse, but that is outside PreRunE scope.)
	if err != nil && strings.Contains(err.Error(), "not found") {
		t.Errorf("existing file should not produce a not-found error, got: %v", err)
	}
}

// ── --out-json directory guard ────────────────────────────────────────────────

// TestProfilePreRunE_OutJSON_MissingDirectory verifies that specifying an
// --out-json path whose directory does not exist is caught early.
func TestProfilePreRunE_OutJSON_MissingDirectory(t *testing.T) {
	t.Cleanup(resetProfileFlags)
	profileXdr = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="
	profileNetwork = "testnet"
	profileOutJSON = "/nonexistent/dir/report.json"

	err := profileCmd.PreRunE(profileCmd, []string{})
	if err == nil {
		t.Fatal("expected error for non-existent --out-json directory")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention the bad directory, got: %q", err.Error())
	}
}

// TestProfilePreRunE_OutJSON_ExistingDirectory_Passes verifies that
// --out-json with a real output directory passes PreRunE.
func TestProfilePreRunE_OutJSON_ExistingDirectory_Passes(t *testing.T) {
	t.Cleanup(resetProfileFlags)
	profileXdr = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="
	profileNetwork = "testnet"
	profileOutJSON = filepath.Join(t.TempDir(), "report.json")

	// PreRunE should not error on the directory check.
	err := profileCmd.PreRunE(profileCmd, []string{})
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		t.Errorf("valid out-json directory should not fail PreRunE, got: %v", err)
	}
}

// TestProfilePreRunE_OutJSON_CurrentDir_Passes verifies that an --out-json
// path in the current directory (no directory component) passes.
func TestProfilePreRunE_OutJSON_CurrentDir_Passes(t *testing.T) {
	t.Cleanup(resetProfileFlags)
	profileXdr = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="
	profileNetwork = "testnet"
	profileOutJSON = "report.json" // relative, current dir

	err := profileCmd.PreRunE(profileCmd, []string{})
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		t.Errorf("current-dir out-json path should not fail PreRunE, got: %v", err)
	}
}

// ── --output (pprof path) validation ─────────────────────────────────────────

// TestProfilePreRunE_Output_DirectoryPath_Rejected verifies that a pprof
// --output path ending with "/" is caught early with a clear error.
func TestProfilePreRunE_Output_DirectoryPath_Rejected(t *testing.T) {
	t.Cleanup(resetProfileFlags)
	profileTraceFile = "" // will be filled via positional arg below
	profileOutput = "./profiles/" // directory path

	tmp := filepath.Join(t.TempDir(), "trace.json")
	if err := os.WriteFile(tmp, []byte(`{"states":[]}`), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	err := profileCmd.PreRunE(profileCmd, []string{tmp})
	if err == nil {
		t.Fatal("expected error for --output that looks like a directory")
	}
	msg := err.Error()
	if !strings.Contains(msg, "directory path") {
		t.Errorf("error should mention 'directory path', got: %q", msg)
	}
	if !strings.Contains(msg, "--output") {
		t.Errorf("error should mention '--output', got: %q", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %q", msg)
	}
}

// TestProfilePreRunE_Output_MissingParentDirectory_Rejected verifies that a
// non-existent parent directory for --output is caught before any work begins.
func TestProfilePreRunE_Output_MissingParentDirectory_Rejected(t *testing.T) {
	t.Cleanup(resetProfileFlags)
	profileOutput = "/nonexistent/subdir/gas.pb.gz"

	tmp := filepath.Join(t.TempDir(), "trace.json")
	if err := os.WriteFile(tmp, []byte(`{"states":[]}`), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	err := profileCmd.PreRunE(profileCmd, []string{tmp})
	if err == nil {
		t.Fatal("expected error for --output with non-existent parent directory")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error should mention 'does not exist', got: %q", err.Error())
	}
}

// TestProfilePreRunE_Output_ExistingDirectory_Passes verifies that a well-formed
// --output path in an existing directory passes PreRunE.
func TestProfilePreRunE_Output_ExistingDirectory_Passes(t *testing.T) {
	t.Cleanup(resetProfileFlags)
	profileOutput = filepath.Join(t.TempDir(), "gas.pb.gz")

	tmp := filepath.Join(t.TempDir(), "trace.json")
	if err := os.WriteFile(tmp, []byte(`{"states":[]}`), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	err := profileCmd.PreRunE(profileCmd, []string{tmp})
	if err != nil && (strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "directory path")) {
		t.Errorf("valid --output path should not fail PreRunE, got: %v", err)
	}
}

// TestProfilePreRunE_Output_Default_Passes verifies that the default
// "profile.pb.gz" output value (set by init) does not trigger directory validation.
func TestProfilePreRunE_Output_Default_Passes(t *testing.T) {
	t.Cleanup(resetProfileFlags)
	// profileOutput is already "profile.pb.gz" from resetProfileFlags

	tmp := filepath.Join(t.TempDir(), "trace.json")
	if err := os.WriteFile(tmp, []byte(`{"states":[]}`), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	err := profileCmd.PreRunE(profileCmd, []string{tmp})
	if err != nil && strings.Contains(err.Error(), "--output") {
		t.Errorf("default --output value should not fail PreRunE, got: %v", err)
	}
}

// ── --out-json path validation (enhanced) ────────────────────────────────────

// TestProfilePreRunE_OutJSON_DirectoryPath_Rejected verifies that an
// --out-json path ending with "/" is caught early with a clear error.
func TestProfilePreRunE_OutJSON_DirectoryPath_Rejected(t *testing.T) {
	t.Cleanup(resetProfileFlags)
	profileXdr = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="
	profileNetwork = "testnet"
	profileOutJSON = "./reports/" // directory path

	err := profileCmd.PreRunE(profileCmd, []string{})
	if err == nil {
		t.Fatal("expected error for --out-json that looks like a directory")
	}
	msg := err.Error()
	if !strings.Contains(msg, "directory path") {
		t.Errorf("error should mention 'directory path', got: %q", msg)
	}
	if !strings.Contains(msg, "--out-json") {
		t.Errorf("error should mention '--out-json', got: %q", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %q", msg)
	}
}

// TestProfilePreRunE_OutJSON_MissingDirectory_HasFixHint verifies that the
// error for a non-existent --out-json directory includes a 'Fix:' hint.
func TestProfilePreRunE_OutJSON_MissingDirectory_HasFixHint(t *testing.T) {
	t.Cleanup(resetProfileFlags)
	profileXdr = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="
	profileNetwork = "testnet"
	profileOutJSON = "/nonexistent/dir/report.json"

	err := profileCmd.PreRunE(profileCmd, []string{})
	if err == nil {
		t.Fatal("expected error for non-existent --out-json directory")
	}
	if !strings.Contains(err.Error(), "Fix:") {
		t.Errorf("error should include a Fix hint, got: %q", err.Error())
	}
}
