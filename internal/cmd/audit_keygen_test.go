// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetKeygenFlags() {
	keygenOutputDir = "."
	keygenKeyName = "audit-signing-key"
	keygenRotate = false
	keygenForce = false
}

func TestAuditKeygen_GeneratesFiles(t *testing.T) {
	dir := t.TempDir()
	resetKeygenFlags()
	keygenOutputDir = dir
	keygenKeyName = "test-key"

	var out bytes.Buffer
	auditKeygenCmd.SetOut(&out)

	if err := auditKeygenCmd.RunE(auditKeygenCmd, nil); err != nil {
		t.Fatalf("RunE failed: %v", err)
	}

	privPath := filepath.Join(dir, "test-key.private.pem")
	pubPath := filepath.Join(dir, "test-key.public.pem")

	if _, err := os.Stat(privPath); err != nil {
		t.Errorf("private key file missing: %v", err)
	}
	if _, err := os.Stat(pubPath); err != nil {
		t.Errorf("public key file missing: %v", err)
	}
}

func TestAuditKeygen_PrivateKeyIsValidPKCS8(t *testing.T) {
	dir := t.TempDir()
	resetKeygenFlags()
	keygenOutputDir = dir
	keygenKeyName = "pkcs8-test"

	if err := auditKeygenCmd.RunE(auditKeygenCmd, nil); err != nil {
		t.Fatalf("RunE failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "pkcs8-test.private.pem"))
	if err != nil {
		t.Fatalf("read private key: %v", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("failed to decode PEM block")
	}
	if block.Type != "PRIVATE KEY" {
		t.Errorf("expected PRIVATE KEY, got %s", block.Type)
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("PKCS8 parse failed: %v", err)
	}
	if _, ok := key.(ed25519.PrivateKey); !ok {
		t.Error("expected Ed25519 private key")
	}
}

func TestAuditKeygen_PublicKeyIsValidSPKI(t *testing.T) {
	dir := t.TempDir()
	resetKeygenFlags()
	keygenOutputDir = dir
	keygenKeyName = "spki-test"

	if err := auditKeygenCmd.RunE(auditKeygenCmd, nil); err != nil {
		t.Fatalf("RunE failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "spki-test.public.pem"))
	if err != nil {
		t.Fatalf("read public key: %v", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("failed to decode PEM block")
	}
	if block.Type != "PUBLIC KEY" {
		t.Errorf("expected PUBLIC KEY, got %s", block.Type)
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatalf("SPKI parse failed: %v", err)
	}
	if _, ok := key.(ed25519.PublicKey); !ok {
		t.Error("expected Ed25519 public key")
	}
}

func TestAuditKeygen_KeyPairIsConsistent(t *testing.T) {
	dir := t.TempDir()
	resetKeygenFlags()
	keygenOutputDir = dir
	keygenKeyName = "pair-test"

	if err := auditKeygenCmd.RunE(auditKeygenCmd, nil); err != nil {
		t.Fatalf("RunE failed: %v", err)
	}

	privData, _ := os.ReadFile(filepath.Join(dir, "pair-test.private.pem"))
	pubData, _ := os.ReadFile(filepath.Join(dir, "pair-test.public.pem"))

	privBlock, _ := pem.Decode(privData)
	privKeyRaw, _ := x509.ParsePKCS8PrivateKey(privBlock.Bytes)
	privKey := privKeyRaw.(ed25519.PrivateKey)

	pubBlock, _ := pem.Decode(pubData)
	pubKeyRaw, _ := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	pubKey := pubKeyRaw.(ed25519.PublicKey)

	message := []byte("glassbox audit test")
	sig := ed25519.Sign(privKey, message)
	if !ed25519.Verify(pubKey, message, sig) {
		t.Error("public key cannot verify signature from private key")
	}
}

func TestAuditKeygen_RejectsExistingFiles(t *testing.T) {
	dir := t.TempDir()
	resetKeygenFlags()
	keygenOutputDir = dir
	keygenKeyName = "existing"

	// First generation — should succeed.
	if err := auditKeygenCmd.RunE(auditKeygenCmd, nil); err != nil {
		t.Fatalf("first generation failed: %v", err)
	}

	// Second generation without --force — should fail.
	if err := auditKeygenCmd.RunE(auditKeygenCmd, nil); err == nil {
		t.Error("expected error when key files already exist")
	}
}

func TestAuditKeygen_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	resetKeygenFlags()
	keygenOutputDir = dir
	keygenKeyName = "force-test"

	if err := auditKeygenCmd.RunE(auditKeygenCmd, nil); err != nil {
		t.Fatalf("first generation failed: %v", err)
	}

	keygenForce = true
	if err := auditKeygenCmd.RunE(auditKeygenCmd, nil); err != nil {
		t.Errorf("force overwrite failed: %v", err)
	}
}

func TestAuditKeygen_RotateFlag_PrintsNote(t *testing.T) {
	dir := t.TempDir()
	resetKeygenFlags()
	keygenOutputDir = dir
	keygenKeyName = "rotate-test"
	keygenRotate = true

	var out bytes.Buffer
	auditKeygenCmd.SetOut(&out)

	if err := auditKeygenCmd.RunE(auditKeygenCmd, nil); err != nil {
		t.Fatalf("RunE failed: %v", err)
	}

	if !strings.Contains(out.String(), "Rotation note") {
		t.Error("expected rotation note in output")
	}
}

func TestMarshalPrivateKeyPEM_RoundTrip(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := marshalPrivateKeyPEM(priv)
	if err != nil {
		t.Fatalf("marshalPrivateKeyPEM: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("nil PEM block")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("parse PKCS8: %v", err)
	}
	if _, ok := parsed.(ed25519.PrivateKey); !ok {
		t.Error("wrong key type")
	}
}

func TestMarshalPublicKeyPEM_RoundTrip(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := marshalPublicKeyPEM(pub)
	if err != nil {
		t.Fatalf("marshalPublicKeyPEM: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("nil PEM block")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatalf("parse SPKI: %v", err)
	}
	if _, ok := parsed.(ed25519.PublicKey); !ok {
		t.Error("wrong key type")
	}
}
