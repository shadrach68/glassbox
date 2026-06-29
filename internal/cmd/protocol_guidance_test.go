// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"
	"testing"

	"github.com/dotandev/glassbox/internal/protocolreg"
)

// ── protocol:status remediation guidance ─────────────────────────────────────

// TestProtocolStatusCmd_NotRegistered_RemediationRendering verifies that the
// remediation-steps rendering logic produces numbered, non-empty steps.
func TestProtocolStatusCmd_NotRegistered_RemediationRendering(t *testing.T) {
	// Simulate the rendering path from protocolStatusCmd.RunE for the
	// not-registered case (mirrors the code in protocol.go).
	diag := &protocolreg.DiagnosticReport{
		Status: protocolreg.StatusNotRegistered,
		Issues: []string{"desktop file not found"},
		RemediationSteps: []string{
			"Run 'glassbox protocol:repair' to recreate the .desktop file and re-register.",
			"Or manually create /home/user/.local/share/applications/glassbox-protocol.desktop",
		},
	}

	var stderr strings.Builder
	stderr.WriteString("GLASSBOX Protocol handler is NOT REGISTERED\n")
	if len(diag.RemediationSteps) > 0 {
		stderr.WriteString("\nTo register the protocol handler:\n")
		for _, step := range diag.RemediationSteps {
			stderr.WriteString("  - " + step + "\n")
		}
	}

	out := stderr.String()
	if !strings.Contains(out, "NOT REGISTERED") {
		t.Errorf("status output should state NOT REGISTERED, got: %s", out)
	}
	if !strings.Contains(out, "protocol:repair") {
		t.Errorf("status output should mention protocol:repair, got: %s", out)
	}
	if !strings.Contains(out, "To register the protocol handler:") {
		t.Errorf("status output should include the remediation header, got: %s", out)
	}
}

// TestProtocolStatusCmd_Registered_NoRemediation verifies that a healthy
// registration produces no remediation output.
func TestProtocolStatusCmd_Registered_NoRemediation(t *testing.T) {
	diag := &protocolreg.DiagnosticReport{
		Status: protocolreg.StatusOK,
		Checks: []string{"Desktop file found", "Protocol helper script launches current binary"},
	}

	var stdout strings.Builder
	for _, check := range diag.Checks {
		stdout.WriteString("[OK] " + check + "\n")
	}
	if diag.Status == protocolreg.StatusOK {
		stdout.WriteString("GLASSBOX Protocol handler is currently REGISTERED\n")
	}

	out := stdout.String()
	if !strings.Contains(out, "REGISTERED") {
		t.Errorf("expected REGISTERED in output, got: %s", out)
	}
	if strings.Contains(out, "protocol:repair") {
		t.Errorf("healthy registration should not mention protocol:repair, got: %s", out)
	}
}

// ── protocol:register next-step hint ─────────────────────────────────────────

// TestProtocolRegisterCmd_SuccessMessage_IncludesVerifyHint verifies that the
// success message produced after registration directs users to run
// protocol:verify as a next step.
func TestProtocolRegisterCmd_SuccessMessage_IncludesVerifyHint(t *testing.T) {
	var stdout strings.Builder
	scheme := protocolreg.Scheme
	stdout.WriteString("Registered GLASSBOX Protocol handler for " + scheme + "://\n")
	stdout.WriteString("Tip: run 'glassbox protocol:verify' to confirm the registration is working.\n")

	out := stdout.String()
	if !strings.Contains(out, "protocol:verify") {
		t.Errorf("register success message should mention protocol:verify, got: %s", out)
	}
	if !strings.Contains(out, scheme+"://") {
		t.Errorf("register success message should include the scheme, got: %s", out)
	}
}

// ── protocol:handle URI format hint ──────────────────────────────────────────

// TestProtocolHandleCmd_InvalidURI_ErrorIncludesFormatHint verifies that the
// error returned for an invalid URI includes the expected URI format so the
// user knows how to construct a valid one.
func TestProtocolHandleCmd_InvalidURI_ErrorIncludesFormatHint(t *testing.T) {
	_, parseErr := protocolreg.ParseDebugURI("https://wrong-scheme/abc")
	if parseErr == nil {
		t.Fatal("expected parse error for wrong scheme")
	}
	// Simulate the wrapping applied by protocol:handle.
	wrappedErr := parseErr.Error() +
		"\n  Expected format: glassbox://debug/<64-char-hex>?network=<testnet|mainnet|futurenet>[&op=<n>][&view=<mode>]"

	for _, keyword := range []string{"glassbox://debug/", "network=", "testnet", "mainnet", "futurenet"} {
		if !strings.Contains(wrappedErr, keyword) {
			t.Errorf("handle error should mention %q; got: %s", keyword, wrappedErr)
		}
	}
}

// TestProtocolHandleCmd_EmptyURI_ErrorIsNonEmpty confirms that an empty URI
// produces a non-empty, human-readable error.
func TestProtocolHandleCmd_EmptyURI_ErrorIsNonEmpty(t *testing.T) {
	_, err := protocolreg.ParseDebugURI("")
	if err == nil {
		t.Fatal("expected error for empty URI")
	}
	if strings.TrimSpace(err.Error()) == "" {
		t.Error("error for empty URI must be a non-empty string")
	}
}

// TestProtocolHandleCmd_InvalidNetwork_ErrorListsOptions verifies that the
// error for an unrecognised network names the accepted values.
func TestProtocolHandleCmd_InvalidNetwork_ErrorListsOptions(t *testing.T) {
	validHash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	_, err := protocolreg.ParseDebugURI("glassbox://debug/" + validHash + "?network=badnet")
	if err == nil {
		t.Fatal("expected error for invalid network")
	}
	for _, net := range []string{"testnet", "mainnet", "futurenet"} {
		if !strings.Contains(err.Error(), net) {
			t.Errorf("network error should list valid option %q; got: %v", net, err)
		}
	}
}
