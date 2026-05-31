// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ContractEvent holds a decoded Soroban contract event and its correlation to
// the call stack at the point it was emitted.
type ContractEvent struct {
	// ContractID is the emitting contract (empty for system events).
	ContractID string `json:"contract_id,omitempty"`
	// Topics contains the decoded event topics as human-readable strings.
	Topics []string `json:"topics"`
	// Data is the decoded event body payload.
	Data string `json:"data"`
	// Type distinguishes "contract", "system", and "diagnostic" events.
	Type string `json:"type"`
	// TraceStep is the index into ExecutionTrace.States where this event
	// was correlated (−1 when correlation is unavailable).
	TraceStep int `json:"trace_step"`
	// CallStack captures the function call path at the point of emission.
	CallStack []string `json:"call_stack,omitempty"`
}

// rawEventEnvelope is used to attempt JSON parsing of an event string.
type rawEventEnvelope struct {
	ContractID string   `json:"contract_id"`
	Topics     []string `json:"topics"`
	Data       interface{} `json:"data"`
	Type       string   `json:"type"`
}

// DecodeContractEvent parses a single raw event string (JSON or structured
// text) into a ContractEvent. It never returns an error; unparseable input is
// preserved verbatim in the Data field.
func DecodeContractEvent(raw string) *ContractEvent {
	raw = strings.TrimSpace(raw)
	ev := &ContractEvent{TraceStep: -1, Type: "contract"}

	// Try JSON first (the canonical Soroban diagnostic event format).
	var envelope rawEventEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err == nil {
		ev.ContractID = envelope.ContractID
		ev.Topics = envelope.Topics
		if envelope.Type != "" {
			ev.Type = envelope.Type
		}
		if envelope.Data != nil {
			switch v := envelope.Data.(type) {
			case string:
				ev.Data = v
			default:
				b, _ := json.Marshal(v)
				ev.Data = string(b)
			}
		}
		return ev
	}

	// Fallback: parse the key:value text format used by the Soroban simulator.
	ev.Data = raw
	for _, field := range strings.Split(raw, " ") {
		kv := strings.SplitN(field, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key, val := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch key {
		case "contract":
			ev.ContractID = val
		case "topic":
			ev.Topics = append(ev.Topics, val)
		case "type":
			ev.Type = val
		case "data":
			ev.Data = val
		}
	}
	return ev
}

// DecodeContractEvents decodes all raw event strings in the slice.
func DecodeContractEvents(raws []string) []*ContractEvent {
	out := make([]*ContractEvent, 0, len(raws))
	for _, r := range raws {
		out = append(out, DecodeContractEvent(r))
	}
	return out
}

// CorrelateEvents matches decoded events to trace steps by contract ID and
// populates each event's TraceStep and CallStack fields. It modifies events
// in place and returns the same slice for convenience.
//
// Matching strategy: iterate states in order; for each event, find the
// nearest state whose ContractID matches (or the first state if no match
// exists). This is a best-effort heuristic because Soroban event ordering
// follows ledger close, not per-call ordering.
func CorrelateEvents(events []*ContractEvent, trace *ExecutionTrace) []*ContractEvent {
	if trace == nil || len(events) == 0 {
		return events
	}

	// Build a quick index: contractID → list of step indices.
	byContract := make(map[string][]int, len(trace.States))
	for i, s := range trace.States {
		if s.ContractID != "" {
			byContract[s.ContractID] = append(byContract[s.ContractID], i)
		}
	}

	// Assign each event to its best matching step.
	// We advance a cursor per contract so each event gets a unique step.
	cursor := make(map[string]int)
	for _, ev := range events {
		steps, ok := byContract[ev.ContractID]
		if !ok || len(steps) == 0 {
			// No matching contract – correlate to the most recent state.
			if len(trace.States) > 0 {
				ev.TraceStep = len(trace.States) - 1
			}
			continue
		}
		idx := cursor[ev.ContractID]
		if idx >= len(steps) {
			idx = len(steps) - 1
		}
		ev.TraceStep = steps[idx]
		cursor[ev.ContractID] = idx + 1

		// Capture a synthetic call stack from ancestor nodes.
		ev.CallStack = buildCallStack(trace, ev.TraceStep)
	}
	return events
}

// buildCallStack reconstructs a call path up to the given step by collecting
// the function names from states at the same or lower depth.
func buildCallStack(trace *ExecutionTrace, step int) []string {
	if step < 0 || step >= len(trace.States) {
		return nil
	}
	var stack []string
	for i := 0; i <= step; i++ {
		s := trace.States[i]
		if s.Function != "" {
			entry := s.Function
			if s.ContractID != "" {
				entry = fmt.Sprintf("%s::%s", s.ContractID[:min(8, len(s.ContractID))], s.Function)
			}
			stack = append(stack, entry)
		}
	}
	// Keep only the last 8 frames to avoid noise.
	if len(stack) > 8 {
		stack = stack[len(stack)-8:]
	}
	return stack
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AttachEventsToNode walks the trace node tree and attaches decoded events to
// the TraceNode whose ContractID matches the event. This enriches the
// interactive viewer with inline event data.
func AttachEventsToNode(root *TraceNode, events []*ContractEvent) {
	if root == nil || len(events) == 0 {
		return
	}
	// Build a map: contractID → events to add.
	byContract := make(map[string][]*ContractEvent, len(events))
	for _, ev := range events {
		byContract[ev.ContractID] = append(byContract[ev.ContractID], ev)
	}
	attachRecursive(root, byContract)
}

func attachRecursive(node *TraceNode, byContract map[string][]*ContractEvent) {
	if node == nil {
		return
	}
	if evs, ok := byContract[node.ContractID]; ok {
		for _, ev := range evs {
			child := NewTraceNode(
				fmt.Sprintf("evt-%s-%d", node.ID, ev.TraceStep),
				"contract_event",
			)
			child.ContractID = ev.ContractID
			child.EventData = formatEventSummary(ev)
			node.AddChild(child)
		}
		// Prevent duplicate attachment on subsequent calls.
		delete(byContract, node.ContractID)
	}
	for _, c := range node.Children {
		attachRecursive(c, byContract)
	}
}

func formatEventSummary(ev *ContractEvent) string {
	if len(ev.Topics) > 0 {
		return fmt.Sprintf("[%s] topics=%s data=%s", ev.Type, strings.Join(ev.Topics, ","), ev.Data)
	}
	return fmt.Sprintf("[%s] %s", ev.Type, ev.Data)
}
