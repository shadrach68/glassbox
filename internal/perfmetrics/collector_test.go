// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package perfmetrics

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollector_RecordRPC(t *testing.T) {
	c := NewCollector()
	c.RecordRPC("getTransaction", 120*time.Millisecond, false)
	c.RecordRPC("getLedgerEntries", 80*time.Millisecond, false)
	c.RecordRPC("simulateTransaction", 200*time.Millisecond, true)

	s := c.Summarize()
	assert.Equal(t, 3, s.RPCCalls)
	assert.Equal(t, 1, s.RPCErrors)
	assert.Equal(t, 400*time.Millisecond, s.RPCTotal)
	assert.Equal(t, 80*time.Millisecond, s.RPCMin)
	assert.Equal(t, 200*time.Millisecond, s.RPCMax)
}

func TestCollector_SimTiming(t *testing.T) {
	c := NewCollector()
	c.StartSim()
	time.Sleep(5 * time.Millisecond)
	c.StopSim()

	s := c.Summarize()
	assert.True(t, s.SimRecorded)
	assert.True(t, s.SimDuration >= 5*time.Millisecond)
}

func TestCollector_StopSim_WithoutStart(t *testing.T) {
	c := NewCollector()
	c.StopSim() // should not panic
	s := c.Summarize()
	assert.False(t, s.SimRecorded)
}

func TestCollector_Print(t *testing.T) {
	c := NewCollector()
	c.RecordRPC("getTransaction", 100*time.Millisecond, false)
	c.RecordRPC("getLedgerEntries", 50*time.Millisecond, true)
	c.StartSim()
	c.StopSim()

	var buf bytes.Buffer
	c.Print(&buf)
	out := buf.String()

	assert.True(t, strings.Contains(out, "RPC calls"), "should contain RPC calls line")
	assert.True(t, strings.Contains(out, "error"), "should mention errors")
	assert.True(t, strings.Contains(out, "Replay time"), "should contain replay time")
}

func TestCollector_Print_NoRPC(t *testing.T) {
	c := NewCollector()
	var buf bytes.Buffer
	c.Print(&buf)
	out := buf.String()
	assert.True(t, strings.Contains(out, "RPC calls"), "header should still appear")
	assert.False(t, strings.Contains(out, "RPC total"), "no total when no calls")
}

// ── New tests for Part C (performance & profiling improvements) ───────────────

// TestCollector_PerMethodBreakdown ensures that when multiple distinct RPC
// methods are recorded the Summary.ByMethod slice is populated and sorted by
// total duration descending.
func TestCollector_PerMethodBreakdown(t *testing.T) {
	c := NewCollector()
	c.RecordRPC("getTransaction", 300*time.Millisecond, false)
	c.RecordRPC("getTransaction", 200*time.Millisecond, false)
	c.RecordRPC("getLedgerEntries", 50*time.Millisecond, false)

	s := c.Summarize()
	require.Len(t, s.ByMethod, 2, "two distinct methods should produce two entries")

	// getTransaction has higher total (500ms) so it must be first.
	assert.Equal(t, "getTransaction", s.ByMethod[0].Method)
	assert.Equal(t, 2, s.ByMethod[0].Calls)
	assert.Equal(t, 500*time.Millisecond, s.ByMethod[0].Total)

	assert.Equal(t, "getLedgerEntries", s.ByMethod[1].Method)
	assert.Equal(t, 1, s.ByMethod[1].Calls)
}

// TestCollector_PerMethodBreakdown_Single verifies that a single RPC method
// does NOT generate a per-method breakdown (redundant with the aggregate totals).
func TestCollector_PerMethodBreakdown_Single(t *testing.T) {
	c := NewCollector()
	c.RecordRPC("getTransaction", 100*time.Millisecond, false)
	c.RecordRPC("getTransaction", 150*time.Millisecond, false)

	s := c.Summarize()
	assert.Empty(t, s.ByMethod, "single method should not produce per-method breakdown")
}

// TestCollector_MethodAvg verifies the Avg helper on MethodSummary.
func TestCollector_MethodAvg(t *testing.T) {
	m := MethodSummary{Calls: 4, Total: 400 * time.Millisecond}
	assert.Equal(t, 100*time.Millisecond, m.Avg())
}

// TestCollector_MethodAvg_ZeroCalls ensures Avg does not divide by zero.
func TestCollector_MethodAvg_ZeroCalls(t *testing.T) {
	m := MethodSummary{Calls: 0, Total: 0}
	assert.Equal(t, time.Duration(0), m.Avg())
}

// TestCollector_SlowCallDetection verifies that calls exceeding SlowRPCThreshold
// are captured in Summary.SlowCalls.
func TestCollector_SlowCallDetection(t *testing.T) {
	oldThreshold := SlowRPCThreshold
	SlowRPCThreshold = 200 * time.Millisecond
	t.Cleanup(func() { SlowRPCThreshold = oldThreshold })

	c := NewCollector()
	c.RecordRPC("getTransaction", 100*time.Millisecond, false)  // fast — not slow
	c.RecordRPC("getLedgerEntries", 250*time.Millisecond, false) // slow
	c.RecordRPC("simulateTransaction", 300*time.Millisecond, true) // slow + error

	s := c.Summarize()
	require.Len(t, s.SlowCalls, 2)
	assert.Equal(t, "getLedgerEntries", s.SlowCalls[0].Method)
	assert.Equal(t, "simulateTransaction", s.SlowCalls[1].Method)
	assert.True(t, s.SlowCalls[1].Err)
}

// TestCollector_SlowCallWarning verifies the Print output includes a warning
// and a remediation tip when slow calls are detected.
func TestCollector_SlowCallWarning(t *testing.T) {
	oldThreshold := SlowRPCThreshold
	SlowRPCThreshold = 50 * time.Millisecond
	t.Cleanup(func() { SlowRPCThreshold = oldThreshold })

	c := NewCollector()
	c.RecordRPC("getTransaction", 100*time.Millisecond, false)

	var buf bytes.Buffer
	c.Print(&buf)
	out := buf.String()

	assert.Contains(t, out, "Slow RPC", "should contain slow RPC warning header")
	assert.Contains(t, out, "getTransaction", "should name the slow method")
	assert.Contains(t, out, "--rpc-url", "should include --rpc-url tip")
}

// TestCollector_Print_PerMethodSection verifies that the per-method section
// appears in the Print output when multiple methods are present.
func TestCollector_Print_PerMethodSection(t *testing.T) {
	c := NewCollector()
	c.RecordRPC("getTransaction", 200*time.Millisecond, false)
	c.RecordRPC("getLedgerEntries", 80*time.Millisecond, false)

	var buf bytes.Buffer
	c.Print(&buf)
	out := buf.String()

	assert.Contains(t, out, "Per-method breakdown")
	assert.Contains(t, out, "getTransaction")
	assert.Contains(t, out, "getLedgerEntries")
}

// TestCollector_PrintJSON_Valid verifies that PrintJSON emits valid JSON with
// the expected top-level fields.
func TestCollector_PrintJSON_Valid(t *testing.T) {
	c := NewCollector()
	c.RecordRPC("getTransaction", 120*time.Millisecond, false)
	c.RecordRPC("getLedgerEntries", 80*time.Millisecond, true)
	c.StartSim()
	c.StopSim()

	var buf bytes.Buffer
	err := c.PrintJSON(&buf)
	require.NoError(t, err)

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))

	assert.EqualValues(t, 2, out["rpc_calls"])
	assert.EqualValues(t, 1, out["rpc_errors"])
	assert.Greater(t, out["rpc_total_ms"], float64(0))
	_, hasSim := out["sim_ms"]
	assert.True(t, hasSim, "sim_ms should be present when simulation was recorded")
}

// TestCollector_PrintJSON_NoRecords verifies that PrintJSON works with an
// empty collector and emits a valid, minimal JSON object.
func TestCollector_PrintJSON_NoRecords(t *testing.T) {
	c := NewCollector()
	var buf bytes.Buffer
	err := c.PrintJSON(&buf)
	require.NoError(t, err)

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))
	assert.EqualValues(t, 0, out["rpc_calls"])
}

// TestCollector_PrintJSON_ByMethod verifies per-method data surfaces in JSON
// when multiple distinct methods are recorded.
func TestCollector_PrintJSON_ByMethod(t *testing.T) {
	c := NewCollector()
	c.RecordRPC("getTransaction", 200*time.Millisecond, false)
	c.RecordRPC("getLedgerEntries", 60*time.Millisecond, false)

	var buf bytes.Buffer
	require.NoError(t, c.PrintJSON(&buf))

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))

	byMethod, ok := out["by_method"].([]interface{})
	require.True(t, ok, "by_method should be a JSON array")
	assert.Len(t, byMethod, 2)
}

// TestCollector_PrintJSON_SlowCalls verifies that slow calls appear in the
// JSON output when they exceed the threshold.
func TestCollector_PrintJSON_SlowCalls(t *testing.T) {
	oldThreshold := SlowRPCThreshold
	SlowRPCThreshold = 100 * time.Millisecond
	t.Cleanup(func() { SlowRPCThreshold = oldThreshold })

	c := NewCollector()
	c.RecordRPC("getTransaction", 500*time.Millisecond, false)

	var buf bytes.Buffer
	require.NoError(t, c.PrintJSON(&buf))

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))

	slow, ok := out["slow_calls"].([]interface{})
	require.True(t, ok, "slow_calls should be present")
	assert.Len(t, slow, 1)
}

// TestCollector_ConcurrentRecording verifies thread safety — multiple goroutines
// recording RPCs concurrently should not race or panic.
func TestCollector_ConcurrentRecording(t *testing.T) {
	c := NewCollector()
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			c.RecordRPC("getTransaction", 10*time.Millisecond, false)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	s := c.Summarize()
	assert.Equal(t, 10, s.RPCCalls)
}
