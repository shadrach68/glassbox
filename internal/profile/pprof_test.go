// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dotandev/glassbox/internal/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceToPprof_NilTrace(t *testing.T) {
	_, err := TraceToPprof(nil)
	assert.Error(t, err)
}

func TestTraceToPprof_EmptyTrace(t *testing.T) {
	execTrace := trace.NewExecutionTrace("tx1", 10)
	p, err := TraceToPprof(execTrace)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Len(t, p.SampleType, 1)
	assert.Equal(t, SampleTypeGas, p.SampleType[0].Type)
	assert.Equal(t, SampleUnitCount, p.SampleType[0].Unit)
	assert.Empty(t, p.Sample)
}

func TestTraceToPprof_WithGas(t *testing.T) {
	execTrace := trace.NewExecutionTrace("tx1", 10)
	execTrace.AddState(trace.ExecutionState{
		Step:       0,
		Operation:  "contract_call",
		ContractID: "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQAHHAGCN4B2",
		Function:   "transfer",
		HostState:  map[string]interface{}{"gas_used": float64(15000)},
	})
	execTrace.AddState(trace.ExecutionState{
		Step:       1,
		Operation:  "contract_call",
		ContractID: "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQAHHAGCN4B2",
		Function:   "mint",
		HostState:  map[string]interface{}{"gas_used": 8000},
	})

	p, err := TraceToPprof(execTrace)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Len(t, p.SampleType, 1)
	assert.Equal(t, SampleTypeGas, p.SampleType[0].Type)
	require.Len(t, p.Sample, 2)
	assert.Equal(t, []int64{15000}, p.Sample[0].Value)
	assert.Equal(t, []int64{8000}, p.Sample[1].Value)
	assert.Len(t, p.Function, 2)
	assert.Len(t, p.Location, 2)
}

func TestTraceToPprof_WritePprof(t *testing.T) {
	execTrace := trace.NewExecutionTrace("tx1", 10)
	execTrace.AddState(trace.ExecutionState{
		Operation:  "contract_call",
		ContractID: "C1",
		Function:   "foo",
		HostState:  map[string]interface{}{"gas_used": int64(1000)},
	})

	var buf bytes.Buffer
	err := WritePprof(execTrace, &buf)
	require.NoError(t, err)
	assert.Greater(t, buf.Len(), 0)
}

func TestTraceToPprof_GasTypes(t *testing.T) {
	execTrace := trace.NewExecutionTrace("tx1", 10)
	execTrace.AddState(trace.ExecutionState{
		Operation: "call", ContractID: "C1", Function: "f",
		HostState: map[string]interface{}{"gas_used": float64(100)},
	})
	execTrace.AddState(trace.ExecutionState{
		Operation: "call", ContractID: "C1", Function: "g",
		HostState: map[string]interface{}{"gas_used": 200},
	})
	execTrace.AddState(trace.ExecutionState{
		Operation: "call", ContractID: "C1", Function: "h",
		HostState: map[string]interface{}{"gas_used": int64(300)},
	})

	p, err := TraceToPprof(execTrace)
	require.NoError(t, err)
	require.Len(t, p.Sample, 3)
	assert.Equal(t, []int64{100}, p.Sample[0].Value)
	assert.Equal(t, []int64{200}, p.Sample[1].Value)
	assert.Equal(t, []int64{300}, p.Sample[2].Value)
}

// ── Diagnostics for empty / step-mismatched traces ───────────────────────────

// TestTraceToPprof_EmptyTrace_ReturnsEmptyProfile verifies that a trace with
// no steps produces a valid but empty pprof profile rather than an error.
// An empty profile is handled gracefully — the caller gets a warning on stderr.
func TestTraceToPprof_EmptyTrace_ReturnsEmptyProfile(t *testing.T) {
	tr := trace.NewExecutionTrace("tx-empty-prof", 10)
	// No states added.

	p, err := TraceToPprof(tr)
	if err != nil {
		t.Fatalf("TraceToPprof should not error on empty trace, got: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil profile even for empty trace")
	}
	if len(p.Sample) != 0 {
		t.Errorf("expected 0 samples for empty trace, got %d", len(p.Sample))
	}
}

// TestTraceToPprof_StepIndexMismatch_ReturnsActionableError verifies that a
// corrupted trace (step index doesn't match position) returns an error with a
// clear Fix hint.
func TestTraceToPprof_StepIndexMismatch_ReturnsActionableError(t *testing.T) {
	tr := trace.NewExecutionTrace("tx-mismatch", 10)
	tr.States = []trace.ExecutionState{
		{Step: 0, Operation: "call", Function: "f"},
		{Step: 5, Operation: "call", Function: "g"}, // wrong index
	}

	_, err := TraceToPprof(tr)
	if err == nil {
		t.Fatal("expected error for step index mismatch")
	}
	msg := err.Error()
	if !strings.Contains(msg, "mismatch") {
		t.Errorf("error should mention mismatch, got: %q", msg)
	}
	// Must include a Fix hint pointing to trace regeneration.
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %q", msg)
	}
	if !strings.Contains(msg, "glassbox debug") {
		t.Errorf("error should mention 'glassbox debug' for regeneration, got: %q", msg)
	}
}

// TestWritePprof_EmptyTrace_Succeeds verifies the full WritePprof pipeline
// handles a zero-step trace without error.
func TestWritePprof_EmptyTrace_Succeeds(t *testing.T) {
	tr := trace.NewExecutionTrace("tx-pprof-empty", 10)

	var buf bytes.Buffer
	err := WritePprof(tr, &buf)
	if err != nil {
		t.Fatalf("WritePprof should succeed for empty trace, got: %v", err)
	}
	// Even an empty pprof profile writes non-zero bytes (header/metadata).
	if buf.Len() == 0 {
		t.Error("pprof output should not be empty even for zero-step trace")
	}
}

// TestTraceToPprof_ZeroGasSteps_ProducesNoSamples verifies that steps with
// zero gas do not produce pprof samples (zero-value samples are skipped).
func TestTraceToPprof_ZeroGasSteps_ProducesNoSamples(t *testing.T) {
	tr := trace.NewExecutionTrace("tx-zero-gas", 10)
	tr.AddState(trace.ExecutionState{
		Step:      0,
		Operation: "call",
		Function:  "f",
		HostState: map[string]interface{}{"gas_used": float64(0)},
	})
	tr.AddState(trace.ExecutionState{
		Step:      1,
		Operation: "call",
		Function:  "g",
		// no gas_used key at all
	})

	p, err := TraceToPprof(tr)
	require.NoError(t, err)
	// Zero-gas steps are filtered out.
	assert.Empty(t, p.Sample, "zero-gas steps should produce no pprof samples")
}

// TestTraceToPprof_MixedGasAndZero_OnlyCapturesPositive verifies that only
// steps with positive gas values contribute to the profile samples.
func TestTraceToPprof_MixedGasAndZero_OnlyCapturesPositive(t *testing.T) {
	tr := trace.NewExecutionTrace("tx-mixed-gas", 10)
	tr.AddState(trace.ExecutionState{
		Step:      0,
		Operation: "call",
		Function:  "expensive",
		HostState: map[string]interface{}{"gas_used": float64(50000)},
	})
	tr.AddState(trace.ExecutionState{
		Step:      1,
		Operation: "call",
		Function:  "free",
		HostState: map[string]interface{}{"gas_used": float64(0)},
	})
	tr.AddState(trace.ExecutionState{
		Step:      2,
		Operation: "call",
		Function:  "cheap",
		HostState: map[string]interface{}{"gas_used": float64(1000)},
	})

	p, err := TraceToPprof(tr)
	require.NoError(t, err)
	assert.Len(t, p.Sample, 2, "only non-zero gas steps should produce samples")
	// Verify the non-zero values are captured correctly.
	gasValues := make([]int64, len(p.Sample))
	for i, s := range p.Sample {
		gasValues[i] = s.Value[0]
	}
	assert.Contains(t, gasValues, int64(50000))
	assert.Contains(t, gasValues, int64(1000))
}
