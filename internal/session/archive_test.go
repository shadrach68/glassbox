// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func sampleData() *Data {
	return &Data{
		ID:              "test-session-01",
		TxHash:          "abcdef1234567890",
		Network:         "testnet",
		Status:          "saved",
		CreatedAt:       time.Now().UTC().Truncate(time.Second),
		LastAccessAt:    time.Now().UTC().Truncate(time.Second),
		EnvelopeXdr:     "AAAA==",
		SimResponseJSON: `{"status":"SUCCESS"}`,
		SchemaVersion:   SchemaVersion,
	}
}

func TestExportAndImportArchive(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "session.gbx")

	original := sampleData()
	if err := ExportArchive(original, archivePath); err != nil {
		t.Fatalf("ExportArchive failed: %v", err)
	}

	// Verify the file exists and is non-empty.
	info, err := os.Stat(archivePath)
	if err != nil {
		t.Fatalf("archive file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("archive file is empty")
	}

	restored, err := ImportArchive(archivePath)
	if err != nil {
		t.Fatalf("ImportArchive failed: %v", err)
	}

	if restored.ID != original.ID {
		t.Errorf("ID mismatch: got %q, want %q", restored.ID, original.ID)
	}
	if restored.TxHash != original.TxHash {
		t.Errorf("TxHash mismatch: got %q, want %q", restored.TxHash, original.TxHash)
	}
	if restored.Network != original.Network {
		t.Errorf("Network mismatch: got %q, want %q", restored.Network, original.Network)
	}
	if restored.EnvelopeXdr != original.EnvelopeXdr {
		t.Errorf("EnvelopeXdr mismatch: got %q, want %q", restored.EnvelopeXdr, original.EnvelopeXdr)
	}
}

func TestExportArchive_NilData(t *testing.T) {
	err := ExportArchive(nil, "/tmp/x.gbx")
	if err == nil {
		t.Fatal("expected error for nil data")
	}
}

func TestExportArchive_EmptyPath(t *testing.T) {
	err := ExportArchive(sampleData(), "")
	if err == nil {
		t.Fatal("expected error for empty dest path")
	}
}

func TestImportArchive_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.gbx")
	if err := os.WriteFile(bad, []byte("not a zip"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ImportArchive(bad)
	if err == nil {
		t.Fatal("expected error for non-zip archive")
	}
}
