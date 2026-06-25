// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"testing"

	"github.com/dotandev/glassbox/internal/simulator"
)

func strPtr(s string) *string { return &s }

func TestBuildExecutionTraceFromSimResponse_Nil(t *testing.T) {
	tr := BuildExecutionTraceFromSimResponse("tx-nil", nil)
	if tr == nil {
		t.Fatal("expected non-nil trace even for nil simResp")
	}
	if len(tr.States) != 0 {
		t.Errorf("expected 0 states for nil simResp, got %d", len(tr.States))
	}
	if tr.TransactionHash != "tx-nil" {
		t.Errorf("unexpected tx hash: %q", tr.TransactionHash)
	}
}

func TestBuildExecutionTraceFromSimResponse_NoEvents(t *testing.T) {
	resp := &simulator.SimulationResponse{Status: "success"}
	tr := BuildExecutionTraceFromSimResponse("tx-empty", resp)
	if tr == nil {
		t.Fatal("expected non-nil trace")
	}
	if len(tr.States) != 0 {
		t.Errorf("expected 0 states, got %d", len(tr.States))
	}
}

func TestBuildExecutionTraceFromSimResponse_WithEvents(t *testing.T) {
	contractID := "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQAHHAGCN4B2"
	resp := &simulator.SimulationResponse{
		Status: "success",
		DiagnosticEvents: []simulator.DiagnosticEvent{
			{
				EventType:  "contract_call",
				ContractID: strPtr(contractID),
				Topics:     []string{"transfer", "100"},
				Data:       "success",
			},
			{
				EventType:  "host_function",
				ContractID: strPtr(contractID),
				Topics:     []string{"balance_of"},
			},
		},
	}

	tr := BuildExecutionTraceFromSimResponse("tx-events", resp)
	if tr == nil {
		t.Fatal("expected non-nil trace")
	}
	if len(tr.States) != 2 {
		t.Fatalf("expected 2 states, got %d", len(tr.States))
	}

	// State 0 checks
	s0 := tr.States[0]
	if s0.EventType != "contract_call" {
		t.Errorf("state 0 EventType: got %q, want %q", s0.EventType, "contract_call")
	}
	if s0.ContractID != contractID {
		t.Errorf("state 0 ContractID: got %q, want %q", s0.ContractID, contractID)
	}
	if len(s0.Arguments) != 2 {
		t.Errorf("state 0 Arguments: expected 2, got %d", len(s0.Arguments))
	}
	if s0.ReturnValue != "success" {
		t.Errorf("state 0 ReturnValue: got %v, want %q", s0.ReturnValue, "success")
	}
	if s0.Step != 0 {
		t.Errorf("state 0 Step should be 0, got %d", s0.Step)
	}

	// State 1 checks
	s1 := tr.States[1]
	if s1.EventType != "host_function" {
		t.Errorf("state 1 EventType: got %q, want %q", s1.EventType, "host_function")
	}
	if s1.Step != 1 {
		t.Errorf("state 1 Step should be 1, got %d", s1.Step)
	}

	// DiagnosticEvents preserved
	if len(tr.DiagnosticEvents) != 2 {
		t.Errorf("expected 2 DiagnosticEvents preserved, got %d", len(tr.DiagnosticEvents))
	}
}

func TestBuildExecutionTraceFromSimResponse_BudgetGasInjected(t *testing.T) {
	contractID := "CTEST"
	resp := &simulator.SimulationResponse{
		Status: "success",
		DiagnosticEvents: []simulator.DiagnosticEvent{
			{EventType: "contract_call", ContractID: strPtr(contractID)},
			{EventType: "host_function", ContractID: strPtr(contractID)},
		},
		BudgetUsage: &simulator.BudgetUsage{
			CPUInstructions: 20000,
			MemoryBytes:     4096,
			OperationsCount: 5,
		},
	}

	tr := BuildExecutionTraceFromSimResponse("tx-budget", resp)
	if len(tr.States) != 2 {
		t.Fatalf("expected 2 states, got %d", len(tr.States))
	}

	for i, state := range tr.States {
		if state.HostState == nil {
			t.Fatalf("state %d: expected HostState to be set, got nil", i)
		}
		gasUsed, ok := state.HostState["gas_used"]
		if !ok {
			t.Fatalf("state %d: missing gas_used in HostState", i)
		}
		f, ok := gasUsed.(float64)
		if !ok {
			t.Fatalf("state %d: gas_used is not float64: %T", i, gasUsed)
		}
		// 20000 / 2 events = 10000.0
		if f != 10000.0 {
			t.Errorf("state %d: gas_used = %f, want 10000.0", i, f)
		}
	}
}

func TestBuildExecutionTraceFromSimResponse_TxHashPreserved(t *testing.T) {
	resp := &simulator.SimulationResponse{
		Status: "success",
		DiagnosticEvents: []simulator.DiagnosticEvent{
			{EventType: "contract_call"},
		},
	}
	const wantHash = "abc123def456"
	tr := BuildExecutionTraceFromSimResponse(wantHash, resp)
	if tr.TransactionHash != wantHash {
		t.Errorf("TransactionHash: got %q, want %q", tr.TransactionHash, wantHash)
	}
}

func TestBuildExecutionTraceFromSimResponse_EndTimeAfterStart(t *testing.T) {
	resp := &simulator.SimulationResponse{
		Status: "success",
		DiagnosticEvents: []simulator.DiagnosticEvent{
			{EventType: "contract_call"},
		},
	}
	tr := BuildExecutionTraceFromSimResponse("tx-timing", resp)
	if !tr.EndTime.IsZero() && tr.EndTime.Before(tr.StartTime) {
		t.Errorf("EndTime %v is before StartTime %v", tr.EndTime, tr.StartTime)
	}
}
