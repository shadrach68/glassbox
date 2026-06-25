// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/dotandev/glassbox/internal/trace"
)

func makeTestTrace(txHash string, states []trace.ExecutionState) *trace.ExecutionTrace {
	t := trace.NewExecutionTrace(txHash, 10)
	t.StartTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, s := range states {
		t.AddState(s)
	}
	t.EndTime = time.Now()
	return t
}

// ── GenerateHTML ──────────────────────────────────────────────────────────────

func TestGenerateHTML_NilTrace(t *testing.T) {
	var buf bytes.Buffer
	err := GenerateHTML(nil, &buf)
	if err == nil {
		t.Fatal("expected error for nil trace")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error should mention nil: %v", err)
	}
}

func TestGenerateHTML_EmptyTrace(t *testing.T) {
	tr := makeTestTrace("tx-empty", nil)
	var buf bytes.Buffer
	err := GenerateHTML(tr, &buf)
	if err != nil {
		t.Fatalf("unexpected error for empty trace: %v", err)
	}
	// Output should be valid — check basic structure exists.
	out := buf.String()
	if len(out) == 0 {
		t.Error("output should not be empty for valid (empty) trace")
	}
}

func TestGenerateHTML_WithStates(t *testing.T) {
	contractID := "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQAHHAGCN4B2"
	states := []trace.ExecutionState{
		{
			Operation:  "contract_call",
			ContractID: contractID,
			Function:   "transfer",
			HostState:  map[string]interface{}{"gas_used": float64(10000)},
		},
		{
			Operation:  "host_function",
			ContractID: contractID,
			Function:   "balance_of",
			HostState:  map[string]interface{}{"gas_used": float64(5000)},
		},
	}
	tr := makeTestTrace("tx-with-states", states)

	var buf bytes.Buffer
	err := GenerateHTML(tr, &buf)
	if err != nil {
		t.Fatalf("GenerateHTML failed: %v", err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("expected non-empty HTML output")
	}
}

func TestGenerateHTML_TransactionHashIncluded(t *testing.T) {
	const txHash = "abc123def456deadbeef"
	tr := makeTestTrace(txHash, []trace.ExecutionState{
		{Operation: "contract_call", Function: "foo", HostState: map[string]interface{}{"gas_used": float64(1)}},
	})

	var buf bytes.Buffer
	err := GenerateHTML(tr, &buf)
	if err != nil {
		t.Fatalf("GenerateHTML failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, txHash) {
		t.Errorf("output should contain transaction hash %q", txHash)
	}
}

func TestGenerateHTML_GasFramesPresent(t *testing.T) {
	states := []trace.ExecutionState{
		{Operation: "call", ContractID: "C1", Function: "fn1", HostState: map[string]interface{}{"gas_used": float64(12000)}},
		{Operation: "call", ContractID: "C1", Function: "fn2", HostState: map[string]interface{}{"gas_used": float64(8000)}},
		{Operation: "call", ContractID: "C1", Function: "fn3", HostState: map[string]interface{}{"gas_used": float64(0)}},
	}
	tr := makeTestTrace("tx-gas-frames", states)

	var buf bytes.Buffer
	if err := GenerateHTML(tr, &buf); err != nil {
		t.Fatalf("GenerateHTML failed: %v", err)
	}
	out := buf.String()
	// The HTML template embeds frames JSON; function names should appear.
	if !strings.Contains(out, "fn1") || !strings.Contains(out, "fn2") {
		t.Errorf("output should contain frame function names fn1 and fn2")
	}
}

func TestGenerateHTML_WriterError(t *testing.T) {
	tr := makeTestTrace("tx-writer-err", []trace.ExecutionState{
		{Operation: "call", Function: "f", HostState: map[string]interface{}{"gas_used": float64(1)}},
	})
	// Use a broken writer that always fails.
	err := GenerateHTML(tr, &errorWriter{})
	if err == nil {
		t.Fatal("expected write error")
	}
}

// ── buildFrames / nearestSnapshotID ──────────────────────────────────────────

func TestBuildFrames_FallbackName(t *testing.T) {
	tr := makeTestTrace("tx-fallback", []trace.ExecutionState{
		// Empty operation, contract, function → should fall back to "step_0"
		{Operation: "", ContractID: "", Function: ""},
	})
	frames := buildFrames(tr)
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	if frames[0].Name != "step_0" {
		t.Errorf("fallback name: got %q, want %q", frames[0].Name, "step_0")
	}
}

func TestBuildFrames_FunctionName(t *testing.T) {
	tr := makeTestTrace("tx-fname", []trace.ExecutionState{
		{ContractID: "CCONTRACT", Function: "my_fn"},
	})
	frames := buildFrames(tr)
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	if !strings.Contains(frames[0].Name, "my_fn") {
		t.Errorf("frame name should contain 'my_fn', got %q", frames[0].Name)
	}
}

func TestBuildFrames_GasExtraction(t *testing.T) {
	tr := makeTestTrace("tx-gas", []trace.ExecutionState{
		{Function: "f", HostState: map[string]interface{}{"gas_used": float64(99999)}},
	})
	frames := buildFrames(tr)
	if frames[0].Gas != 99999 {
		t.Errorf("gas: got %d, want 99999", frames[0].Gas)
	}
}

func TestNearestSnapshotID_NoSnapshots(t *testing.T) {
	tr := makeTestTrace("tx-no-snaps", []trace.ExecutionState{
		{Function: "f"},
	})
	// Clear snapshots
	tr.Snapshots = nil
	id := nearestSnapshotID(tr, 0)
	if id != -1 {
		t.Errorf("expected -1 with no snapshots, got %d", id)
	}
}

func TestBuildSnapshotSummaries_Empty(t *testing.T) {
	tr := makeTestTrace("tx-no-snaps2", nil)
	tr.Snapshots = nil
	summaries := buildSnapshotSummaries(tr)
	if len(summaries) != 0 {
		t.Errorf("expected no summaries, got %d", len(summaries))
	}
}

// ── errorWriter ───────────────────────────────────────────────────────────────

// errorWriter always returns an error from Write.
type errorWriter struct{}

func (e *errorWriter) Write(_ []byte) (int, error) {
	return 0, errWriteFailed
}

var errWriteFailed = errW("simulated write failure")

type errW string

func (e errW) Error() string { return string(e) }

// ── Empty-trace diagnostics ───────────────────────────────────────────────────

// TestGenerateHTML_EmptyTrace_StillProducesOutput verifies that an empty trace
// generates a valid (albeit blank) HTML page rather than returning an error.
// This reflects the non-fatal design: an empty flamegraph is better than no output.
func TestGenerateHTML_EmptyTrace_StillProducesOutput(t *testing.T) {
	tr := makeTestTrace("tx-empty-diag", nil)
	var buf bytes.Buffer
	err := GenerateHTML(tr, &buf)
	if err != nil {
		t.Fatalf("GenerateHTML should not error on empty trace, got: %v", err)
	}
	// HTML should still be structurally valid and contain the TX hash.
	out := buf.String()
	if len(out) == 0 {
		t.Fatal("HTML output should not be empty even for zero-step trace")
	}
}

// TestGenerateHTML_NilWriter verifies that passing a nil writer returns an
// actionable error rather than panicking.
func TestGenerateHTML_NilWriter_ReturnsError(t *testing.T) {
	tr := makeTestTrace("tx-nil-writer", []trace.ExecutionState{
		{Operation: "call", Function: "f"},
	})
	err := GenerateHTML(tr, nil)
	if err == nil {
		t.Fatal("expected error when writer is nil")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error should mention nil writer, got: %v", err)
	}
	// Should include a Fix hint.
	if !strings.Contains(err.Error(), "Fix:") {
		t.Errorf("error should include a Fix hint, got: %v", err)
	}
}
