// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/dotandev/glassbox/internal/trace"
)

//go:embed template.html
var flamegraphTemplate string

// FrameData holds per-frame metadata for the interactive flamegraph.
type FrameData struct {
	Name       string `json:"name"`
	Gas        int64  `json:"gas"`
	Step       int    `json:"step"`
	StateChurn int    `json:"state_churn"`
	SnapshotID int    `json:"snapshot_id"`
}

// SnapshotSummary is a lightweight record of a state snapshot for the flamegraph UI.
type SnapshotSummary struct {
	SnapshotID int `json:"snapshot_id"`
	Step       int `json:"step"`
	KeyCount   int `json:"key_count"`
}

// GenerateHTML writes an interactive flamegraph HTML page to w.
// Each frame carries State Churn (number of HostState keys modified at that step)
// and a Snapshot ID identifying the nearest snapshot, enabling click-to-jump navigation.
func GenerateHTML(execTrace *trace.ExecutionTrace, w io.Writer) error {
	if execTrace == nil {
		return fmt.Errorf("execution trace is nil — cannot generate flamegraph\n" +
			"  Fix: ensure the simulation completed successfully before calling GenerateHTML\n" +
			"  Tip: run 'glassbox debug --profile <tx-hash>' to generate a flamegraph automatically")
	}
	if w == nil {
		return fmt.Errorf("writer is nil — cannot write flamegraph output\n" +
			"  Fix: provide a valid io.Writer (e.g. a file or bytes.Buffer)")
	}

	// Warn when the trace carries no steps — the flamegraph will be empty but
	// the HTML file is still valid (blank canvas).  Surface a hint rather than
	// failing hard so callers can decide how to handle the situation.
	if len(execTrace.States) == 0 {
		fmt.Fprintf(os.Stderr,
			"Warning: trace contains no execution steps — flamegraph will be empty.\n"+
				"  Possible causes:\n"+
				"    - The simulation produced no diagnostic events\n"+
				"    - The trace was captured by an older simulator version\n"+
				"  Tip: use 'glassbox profile --xdr <tx.xdr>' for a live gas report,\n"+
				"       or re-run with 'glassbox debug --save-snapshots <file>'\n",
		)
	}

	frames := buildFrames(execTrace)
	summaries := buildSnapshotSummaries(execTrace)

	framesJSON, err := json.Marshal(frames)
	if err != nil {
		return fmt.Errorf("failed to marshal flamegraph frames: %w\n"+
			"  This may indicate the trace contains non-serializable data (e.g. NaN or Inf gas values)\n"+
			"  Fix: verify the trace was produced by a supported simulator version", err)
	}
	summariesJSON, err := json.Marshal(summaries)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot summaries: %w\n"+
			"  Fix: verify the trace snapshot data is valid", err)
	}

	tmpl, err := template.New("flamegraph").Parse(flamegraphTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse flamegraph template: %w\n"+
			"  This is an internal error — please report it with 'glassbox version' output", err)
	}

	data := map[string]interface{}{
		"Frames":    string(framesJSON),
		"Snapshots": string(summariesJSON),
		"TxHash":    execTrace.TransactionHash,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render flamegraph template: %w\n"+
			"  This is an internal error — please report it with 'glassbox version' output", err)
	}

	_, err = w.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write flamegraph output: %w\n"+
			"  Fix: ensure the output destination is writable and has sufficient space", err)
	}
	return nil
}

// buildFrames constructs a FrameData slice from the execution trace states.
func buildFrames(execTrace *trace.ExecutionTrace) []FrameData {
	frames := make([]FrameData, 0, len(execTrace.States))
	for i := range execTrace.States {
		state := &execTrace.States[i]
		name := functionName(state)
		if name == "" {
			name = state.Operation
		}
		if name == "" {
			name = fmt.Sprintf("step_%d", state.Step)
		}
		frames = append(frames, FrameData{
			Name:       name,
			Gas:        extractGasFromState(state),
			Step:       state.Step,
			StateChurn: len(state.HostState),
			SnapshotID: nearestSnapshotID(execTrace, state.Step),
		})
	}
	return frames
}

// buildSnapshotSummaries converts each snapshot into a SnapshotSummary for the UI.
func buildSnapshotSummaries(execTrace *trace.ExecutionTrace) []SnapshotSummary {
	summaries := make([]SnapshotSummary, 0, len(execTrace.Snapshots))
	for i := range execTrace.Snapshots {
		snap := &execTrace.Snapshots[i]
		summaries = append(summaries, SnapshotSummary{
			SnapshotID: i,
			Step:       snap.Step,
			KeyCount:   len(snap.HostState),
		})
	}
	return summaries
}

// nearestSnapshotID returns the index of the latest snapshot whose Step is <= step.
// Returns -1 when no snapshot exists at or before step.
func nearestSnapshotID(execTrace *trace.ExecutionTrace, step int) int {
	id := -1
	for i, snap := range execTrace.Snapshots {
		if snap.Step <= step {
			id = i
		} else {
			break
		}
	}
	return id
}
