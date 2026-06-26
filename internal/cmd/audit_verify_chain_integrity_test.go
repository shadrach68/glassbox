// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeInvalidHashIntoLog injects a structurally invalid (non-hex) value for
// previous_signature_hash into the provenance of the log at path.
func writeInvalidHashIntoLog(t *testing.T, path, badHash string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var log SignedAuditLog
	require.NoError(t, json.Unmarshal(data, &log))
	log.Provenance = &SignatureProvenance{
		SignerIdentity:        "ci@example.com",
		Algorithm:             "Ed25519",
		PreviousSignatureHash: badHash, // intentionally invalid
	}
	out, err := json.MarshalIndent(log, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, out, 0600))
}

// TestAuditVerify_EmbeddedHashMalformedWithoutFlag checks that when
// previous_signature_hash in the log is structurally invalid (too short / non-hex)
// and --previous-signature-hash is not supplied, the output warns about the
// malformed field rather than silently claiming it as "not verified".
func TestAuditVerify_EmbeddedHashMalformedWithoutFlag(t *testing.T) {
	defer resetAuditVerifyFlags()
	payload := map[string]interface{}{"input": "data"}
	path, _, _ := buildSignedLog(t, payload)
	writeInvalidHashIntoLog(t, path, "this-is-not-valid-hex-and-too-short")

	resetAuditVerifyFlags()
	auditVerifyFile = path
	// No --previous-signature-hash supplied.

	var buf bytes.Buffer
	auditVerifyCmd.SetOut(&buf)
	err := auditVerifyCmd.RunE(auditVerifyCmd, nil)
	require.NoError(t, err) // overall result is still VALID (hash + sig pass)

	out := buf.String()
	// Should warn about the malformed embedded field.
	if !strings.Contains(out, "invalid") && !strings.Contains(out, "malformed") && !strings.Contains(out, "format is invalid") {
		t.Errorf("output should warn about malformed previous_signature_hash, got: %s", out)
	}
	// Should NOT say "not verified" for a malformed field — that would under-report.
	// (The message should be more alarming than the normal unverified note.)
	if strings.Contains(out, "chain linkage was not verified") {
		t.Errorf("malformed hash should produce a stronger warning than the normal 'not verified' note, got: %s", out)
	}
}

// TestAuditVerify_EmbeddedHashValidFormatUnverified verifies the normal case:
// a well-formed embedded hash without --previous-signature-hash produces the
// standard "not verified" note, not a warning about malformation.
func TestAuditVerify_EmbeddedHashValidFormatUnverified(t *testing.T) {
	defer resetAuditVerifyFlags()
	payload := map[string]interface{}{"input": "data"}
	path, _, _ := buildSignedLog(t, payload)
	writePrevHashIntoLog(t, path, validChainHash) // validChainHash = 64 valid hex chars

	resetAuditVerifyFlags()
	auditVerifyFile = path
	// No --previous-signature-hash supplied.

	var buf bytes.Buffer
	auditVerifyCmd.SetOut(&buf)
	err := auditVerifyCmd.RunE(auditVerifyCmd, nil)
	require.NoError(t, err)

	out := buf.String()
	// Should have the standard "not verified" note, not a malformation warning.
	assert.Contains(t, out, "not verified",
		"well-formed unverified hash should show standard 'not verified' note")
	assert.NotContains(t, out, "format is invalid",
		"well-formed hash should not trigger a format-invalid warning")
}

// TestAuditVerify_JSONOutput_ChainNoteIncluded verifies that the chain_note field
// in JSON output carries the correct message for an unverified but well-formed hash.
func TestAuditVerify_JSONOutput_ChainNoteForUnverifiedHash(t *testing.T) {
	defer resetAuditVerifyFlags()
	payload := map[string]interface{}{"input": "data"}
	path, _, _ := buildSignedLog(t, payload)
	writePrevHashIntoLog(t, path, validChainHash)

	resetAuditVerifyFlags()
	auditVerifyFile = path
	auditVerifyJSON = true

	var buf bytes.Buffer
	auditVerifyCmd.SetOut(&buf)
	err := auditVerifyCmd.RunE(auditVerifyCmd, nil)
	require.NoError(t, err)

	var result auditVerifyResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.True(t, result.Valid)
	assert.Nil(t, result.ChainLinkValid, "chain_link_valid should be absent when not verified")
	assert.NotEmpty(t, result.ChainNote, "chain_note should be set when hash present but unverified")
	assert.Contains(t, result.ChainNote, "not verified")
}

// TestAuditVerify_JSONOutput_MalformedHashNote verifies that JSON output carries
// a distinguishable chain_note for a malformed embedded previous_signature_hash.
func TestAuditVerify_JSONOutput_MalformedHashNote(t *testing.T) {
	defer resetAuditVerifyFlags()
	payload := map[string]interface{}{"input": "data"}
	path, _, _ := buildSignedLog(t, payload)
	writeInvalidHashIntoLog(t, path, "tooshort")

	resetAuditVerifyFlags()
	auditVerifyFile = path
	auditVerifyJSON = true

	var buf bytes.Buffer
	auditVerifyCmd.SetOut(&buf)
	err := auditVerifyCmd.RunE(auditVerifyCmd, nil)
	require.NoError(t, err)

	var result auditVerifyResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.True(t, result.Valid, "overall valid should still be true (hash + sig pass)")
	assert.NotEmpty(t, result.ChainNote)
	// The note for a malformed hash should differ from the normal unverified note.
	assert.NotEqual(t, result.ChainNote,
		"previous_signature_hash is present but chain linkage was not verified; "+
			"pass --previous-signature-hash <hex> to confirm this log links to the expected predecessor",
		"malformed hash note should be distinct from the standard unverified note")
}

// TestAuditVerify_ChainLinkPass_WithValidProvenanceHash verifies end-to-end
// that a log with a well-formed provenance hash and matching --previous-signature-hash
// results in [PASS] Chain link.
func TestAuditVerify_ChainLinkPass_WithValidProvenanceHash(t *testing.T) {
	defer resetAuditVerifyFlags()
	payload := map[string]interface{}{"input": "value"}
	path, _, _ := buildSignedLog(t, payload)
	writePrevHashIntoLog(t, path, validChainHash)

	resetAuditVerifyFlags()
	auditVerifyFile = path
	auditVerifyPreviousHash = validChainHash

	var buf bytes.Buffer
	auditVerifyCmd.SetOut(&buf)
	err := auditVerifyCmd.RunE(auditVerifyCmd, nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "[PASS] Chain link")
	assert.Contains(t, buf.String(), "VALID")
}
