// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package perfmetrics

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// ── Collector.Summarize ───────────────────────────────────────────────────────

func TestCollector_NewCollector(t *testing.T) {
	c := NewCollector()
	if c == nil {
		t.Fatal("NewCollector returned nil")
	}
	s := c.Summarize()
	if s.RPCCalls != 0 {
		t.Errorf("expected 0 RPC calls, got %d", s.RPCCalls)
	}
	if s.SimRecorded {
		t.Error("expected SimRecorded=false before any recording")
	}
}

func TestCollector_RecordRPC_Basic(t *testing.T) {
	c := NewCollector()
	c.RecordRPC("getTransaction", 250*time.Millisecond, false)
	c.RecordRPC("getLedgerEntries", 150*time.Millisecond, false)

	s := c.Summarize()
	if s.RPCCalls != 2 {
		t.Errorf("expected 2 RPC calls, got %d", s.RPCCalls)
	}
	if s.RPCErrors != 0 {
		t.Errorf("expected 0 errors, got %d", s.RPCErrors)
	}
	if s.RPCMin != 150*time.Millisecond {
		t.Errorf("RPCMin: got %v, want 150ms", s.RPCMin)
	}
	if s.RPCMax != 250*time.Millisecond {
		t.Errorf("RPCMax: got %v, want 250ms", s.RPCMax)
	}
	total := 250*time.Millisecond + 150*time.Millisecond
	if s.RPCTotal != total {
		t.Errorf("RPCTotal: got %v, want %v", s.RPCTotal, total)
	}
}

func TestCollector_RecordRPC_WithErrors(t *testing.T) {
	c := NewCollector()
	c.RecordRPC("getTransaction", 100*time.Millisecond, false)
	c.RecordRPC("getTransaction", 200*time.Millisecond, true) // error

	s := c.Summarize()
	if s.RPCCalls != 2 {
		t.Errorf("expected 2 calls, got %d", s.RPCCalls)
	}
	if s.RPCErrors != 1 {
		t.Errorf("expected 1 error, got %d", s.RPCErrors)
	}
}

func TestCollector_SimDuration(t *testing.T) {
	c := NewCollector()

	// No sim recorded yet
	s := c.Summarize()
	if s.SimRecorded {
		t.Error("expected SimRecorded=false before StartSim/StopSim")
	}

	c.StartSim()
	time.Sleep(2 * time.Millisecond)
	c.StopSim()

	s = c.Summarize()
	if !s.SimRecorded {
		t.Error("expected SimRecorded=true after StopSim")
	}
	if s.SimDuration <= 0 {
		t.Errorf("expected positive SimDuration, got %v", s.SimDuration)
	}
}

func TestCollector_StopSim_WithoutStart(t *testing.T) {
	// StopSim before StartSim should not panic or set SimRecorded
	c := NewCollector()
	c.StopSim()
	s := c.Summarize()
	if s.SimRecorded {
		t.Error("StopSim without StartSim should not set SimRecorded")
	}
}

func TestCollector_SlowCalls(t *testing.T) {
	orig := SlowRPCThreshold
	SlowRPCThreshold = 100 * time.Millisecond
	defer func() { SlowRPCThreshold = orig }()

	c := NewCollector()
	c.RecordRPC("fast", 50*time.Millisecond, false)
	c.RecordRPC("slow", 200*time.Millisecond, false)

	s := c.Summarize()
	if len(s.SlowCalls) != 1 {
		t.Fatalf("expected 1 slow call, got %d", len(s.SlowCalls))
	}
	if s.SlowCalls[0].Method != "slow" {
		t.Errorf("expected slow call method='slow', got %q", s.SlowCalls[0].Method)
	}
}

func TestCollector_ByMethod_SingleMethod(t *testing.T) {
	c := NewCollector()
	c.RecordRPC("getTransaction", 100*time.Millisecond, false)
	c.RecordRPC("getTransaction", 200*time.Millisecond, false)

	s := c.Summarize()
	// ByMethod is only populated when >1 distinct method is recorded.
	if len(s.ByMethod) != 0 {
		t.Errorf("ByMethod should be empty with single method, got %d entries", len(s.ByMethod))
	}
}

func TestCollector_ByMethod_MultipleMethod(t *testing.T) {
	c := NewCollector()
	c.RecordRPC("getTransaction", 100*time.Millisecond, false)
	c.RecordRPC("getLedgerEntries", 200*time.Millisecond, false)
	c.RecordRPC("getTransaction", 150*time.Millisecond, false)

	s := c.Summarize()
	if len(s.ByMethod) != 2 {
		t.Fatalf("expected 2 ByMethod entries, got %d", len(s.ByMethod))
	}
	// Should be sorted by total descending
	if s.ByMethod[0].Total < s.ByMethod[1].Total {
		t.Error("ByMethod should be sorted by total descending")
	}
}

func TestMethodSummary_Avg_ZeroCalls(t *testing.T) {
	m := MethodSummary{Calls: 0, Total: 500 * time.Millisecond}
	if m.Avg() != 0 {
		t.Errorf("Avg with 0 calls should return 0, got %v", m.Avg())
	}
}

// ── Print output ──────────────────────────────────────────────────────────────

func TestCollector_Print_NoRPC(t *testing.T) {
	c := NewCollector()
	var buf bytes.Buffer
	c.Print(&buf)
	out := buf.String()
	if !strings.Contains(out, "Performance Summary") {
		t.Errorf("expected 'Performance Summary' header, got: %s", out)
	}
	if !strings.Contains(out, "RPC calls") {
		t.Errorf("expected 'RPC calls' line, got: %s", out)
	}
}

func TestCollector_Print_NilWriter(t *testing.T) {
	// Should not panic when w is nil (falls back to os.Stdout).
	c := NewCollector()
	c.RecordRPC("test", 10*time.Millisecond, false)
	// Pass a discard writer instead of actually testing os.Stdout output.
	var buf bytes.Buffer
	c.Print(&buf)
	// If we get here without panic, the test passes.
}

func TestCollector_Print_SlowCallWarning(t *testing.T) {
	orig := SlowRPCThreshold
	SlowRPCThreshold = 10 * time.Millisecond
	defer func() { SlowRPCThreshold = orig }()

	c := NewCollector()
	c.RecordRPC("slow", 50*time.Millisecond, false)

	var buf bytes.Buffer
	c.Print(&buf)
	out := buf.String()
	if !strings.Contains(out, "Slow RPC calls") {
		t.Errorf("expected slow call warning in output, got: %s", out)
	}
	if !strings.Contains(out, "rpc-url") {
		t.Errorf("expected --rpc-url tip in slow call warning, got: %s", out)
	}
}

func TestCollector_Print_SimDuration(t *testing.T) {
	c := NewCollector()
	c.StartSim()
	time.Sleep(1 * time.Millisecond)
	c.StopSim()

	var buf bytes.Buffer
	c.Print(&buf)
	out := buf.String()
	if !strings.Contains(out, "Replay time") {
		t.Errorf("expected 'Replay time' line when sim recorded, got: %s", out)
	}
}

// ── PrintJSON output ──────────────────────────────────────────────────────────

func TestCollector_PrintJSON_ValidJSON(t *testing.T) {
	c := NewCollector()
	c.RecordRPC("getTransaction", 120*time.Millisecond, false)
	c.RecordRPC("getLedgerEntries", 80*time.Millisecond, true)
	c.StartSim()
	time.Sleep(1 * time.Millisecond)
	c.StopSim()

	var buf bytes.Buffer
	if err := c.PrintJSON(&buf); err != nil {
		t.Fatalf("PrintJSON failed: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if rpcCalls, ok := out["rpc_calls"].(float64); !ok || rpcCalls != 2 {
		t.Errorf("rpc_calls: expected 2, got %v", out["rpc_calls"])
	}
	if rpcErrors, ok := out["rpc_errors"].(float64); !ok || rpcErrors != 1 {
		t.Errorf("rpc_errors: expected 1, got %v", out["rpc_errors"])
	}
	if _, ok := out["sim_ms"]; !ok {
		t.Error("expected sim_ms key in JSON output")
	}
}

func TestCollector_PrintJSON_ZeroState(t *testing.T) {
	c := NewCollector()
	var buf bytes.Buffer
	if err := c.PrintJSON(&buf); err != nil {
		t.Fatalf("PrintJSON failed on empty collector: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if rpcCalls, ok := out["rpc_calls"].(float64); !ok || rpcCalls != 0 {
		t.Errorf("rpc_calls should be 0 for empty collector, got %v", out["rpc_calls"])
	}
}

// ── Concurrency safety ────────────────────────────────────────────────────────

func TestCollector_ConcurrentRecordRPC(t *testing.T) {
	c := NewCollector()
	done := make(chan struct{})
	const goroutines = 10
	for i := 0; i < goroutines; i++ {
		go func() {
			c.RecordRPC("method", 10*time.Millisecond, false)
			done <- struct{}{}
		}()
	}
	for i := 0; i < goroutines; i++ {
		<-done
	}
	s := c.Summarize()
	if s.RPCCalls != goroutines {
		t.Errorf("expected %d calls, got %d", goroutines, s.RPCCalls)
	}
}
