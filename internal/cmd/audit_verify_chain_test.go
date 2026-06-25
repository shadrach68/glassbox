// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validChainHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 64 hex

// writePrevHashIntoLog loads the signed log at path, sets its provenance
// previous_signature_hash, and rewrites it. The Ed25519 signature is computed
// over the payload hash, so injecting provenance does not invalidate it.
func writePrevHashIntoLog(t *testing.T, path, prevHash string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var log SignedAuditLog
	require.NoError(t, json.Unmarshal(data, &log))
	log.Provenance = &SignatureProvenance{
		SignerIdentity:        "ci@example.com",
		Algorithm:             "Ed25519",
		PreviousSignatureHash: prevHash,
	}
	out, err := json.MarshalIndent(log, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, out, 0600))
}

func resetAuditVerifyFlags() {
	auditVerifyFile = ""
	auditVerifyPublicKey = ""
	auditVerifySchema = ""
	auditVerifyJSON = false
	auditVerifyPreviousHash = ""
}

func TestValidateAuditVerifyInputs(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	validPub := hex.EncodeToString(pub)

	tests := []struct {
		name       string
		pubKey     string
		schema     string
		prevHash   string
		wantErr    bool
		wantSubstr string
	}{
		{name: "all empty", wantErr: false},
		{name: "valid public key", pubKey: validPub, wantErr: false},
		{name: "non-hex public key", pubKey: "zz", wantErr: true, wantSubstr: "public-key"},
		{name: "wrong-length public key", pubKey: hex.EncodeToString([]byte("short")), wantErr: true, wantSubstr: "public-key"},
		{name: "missing schema file", schema: "C:/no/such/schema.json", wantErr: true, wantSubstr: "schema"},
		{name: "valid previous hash", prevHash: validChainHash, wantErr: false},
		{name: "short previous hash", prevHash: "abc123", wantErr: true, wantSubstr: "previous-signature-hash"},
		{name: "non-hex previous hash", prevHash: strings.Repeat("z", 64), wantErr: true, wantSubstr: "previous-signature-hash"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := validateAuditVerifyInputs(tt.pubKey, tt.schema, tt.prevHash)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantSubstr != "" {
					assert.Contains(t, err.Error(), tt.wantSubstr)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateAuditLogFields(t *testing.T) {
	complete := &SignedAuditLog{
		Signature: strings.Repeat("c", 128),
		TraceHash: validChainHash,
		PublicKey: validChainHash,
		Payload:   json.RawMessage(`{}`),
	}

	t.Run("complete log passes", func(t *testing.T) {
		require.NoError(t, validateAuditLogFields(complete, false))
	})

	t.Run("missing signature", func(t *testing.T) {
		l := *complete
		l.Signature = ""
		err := validateAuditLogFields(&l, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signature")
	})

	t.Run("missing public_key without override", func(t *testing.T) {
		l := *complete
		l.PublicKey = ""
		err := validateAuditLogFields(&l, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "public_key")
	})

	t.Run("missing public_key allowed with override", func(t *testing.T) {
		l := *complete
		l.PublicKey = ""
		require.NoError(t, validateAuditLogFields(&l, true))
	})

	t.Run("missing payload", func(t *testing.T) {
		l := *complete
		l.Payload = nil
		err := validateAuditLogFields(&l, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "payload")
	})
}

func TestVerifyChainLink(t *testing.T) {
	t.Run("matching hash", func(t *testing.T) {
		log := &SignedAuditLog{Provenance: &SignatureProvenance{PreviousSignatureHash: validChainHash}}
		ok, err := verifyChainLink(log, validChainHash)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("case-insensitive match", func(t *testing.T) {
		log := &SignedAuditLog{Provenance: &SignatureProvenance{PreviousSignatureHash: strings.ToUpper(validChainHash)}}
		ok, err := verifyChainLink(log, validChainHash)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("mismatch", func(t *testing.T) {
		log := &SignedAuditLog{Provenance: &SignatureProvenance{PreviousSignatureHash: validChainHash}}
		ok, err := verifyChainLink(log, strings.Repeat("b", 64))
		require.Error(t, err)
		assert.False(t, ok)
		assert.Contains(t, err.Error(), "chain link broken")
	})

	t.Run("no provenance", func(t *testing.T) {
		ok, err := verifyChainLink(&SignedAuditLog{}, validChainHash)
		require.Error(t, err)
		assert.False(t, ok)
		assert.Contains(t, err.Error(), "no previous_signature_hash")
	})

	t.Run("empty previous hash", func(t *testing.T) {
		log := &SignedAuditLog{Provenance: &SignatureProvenance{PreviousSignatureHash: ""}}
		ok, err := verifyChainLink(log, validChainHash)
		require.Error(t, err)
		assert.False(t, ok)
	})
}

func TestAuditVerify_ChainLinkValid(t *testing.T) {
	defer resetAuditVerifyFlags()
	payload := map[string]interface{}{"input": "data"}
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

func TestAuditVerify_ChainLinkBroken(t *testing.T) {
	defer resetAuditVerifyFlags()
	payload := map[string]interface{}{"input": "data"}
	path, _, _ := buildSignedLog(t, payload)
	writePrevHashIntoLog(t, path, validChainHash)

	resetAuditVerifyFlags()
	auditVerifyFile = path
	auditVerifyPreviousHash = strings.Repeat("b", 64) // wrong predecessor

	var buf bytes.Buffer
	auditVerifyCmd.SetOut(&buf)
	err := auditVerifyCmd.RunE(auditVerifyCmd, nil)
	require.Error(t, err)
	assert.Contains(t, buf.String(), "[FAIL] Chain link")
	assert.Contains(t, buf.String(), "INVALID")
}

func TestAuditVerify_ChainHashPresentButUnverified(t *testing.T) {
	defer resetAuditVerifyFlags()
	payload := map[string]interface{}{"input": "data"}
	path, _, _ := buildSignedLog(t, payload)
	writePrevHashIntoLog(t, path, validChainHash)

	resetAuditVerifyFlags()
	auditVerifyFile = path
	// No --previous-signature-hash supplied: linkage must not be silently claimed.

	var buf bytes.Buffer
	auditVerifyCmd.SetOut(&buf)
	err := auditVerifyCmd.RunE(auditVerifyCmd, nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "not verified")
	assert.NotContains(t, buf.String(), "[PASS] Chain link")
}

func TestAuditVerify_PreRunRejectsBadPublicKey(t *testing.T) {
	defer resetAuditVerifyFlags()
	payload := map[string]interface{}{"input": "data"}
	path, _, _ := buildSignedLog(t, payload)

	resetAuditVerifyFlags()
	auditVerifyFile = path
	auditVerifyPublicKey = "not-hex"

	err := auditVerifyCmd.PreRunE(auditVerifyCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "public-key")
}
