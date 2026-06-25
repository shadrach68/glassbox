// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"fmt"
	"io"
	"os"

	"github.com/dotandev/glassbox/internal/trace"
	goprofile "github.com/google/pprof/profile"
)

const (
	// SampleTypeGas is the pprof sample type for gas consumption.
	SampleTypeGas = "gas"
	// SampleUnitCount is the unit for gas samples.
	SampleUnitCount = "count"
)

// TraceToPprof synthesizes an execution trace into a pprof-compliant profile
// that maps gas consumption to functions. The result can be viewed with
// go tool pprof.
func TraceToPprof(execTrace *trace.ExecutionTrace) (*goprofile.Profile, error) {
	if execTrace == nil {
		return nil, fmt.Errorf("execution trace is nil — cannot generate pprof profile\n" +
			"  Fix: ensure the simulation completed successfully before profiling\n" +
			"  Tip: run 'glassbox debug --save-snapshots <file>' to generate a trace")
	}

	// Validate step ordering up-front so we surface corruption early.
	for i, state := range execTrace.States {
		if state.Step != i {
			return nil, fmt.Errorf(
				"trace step index mismatch at position %d: state.Step=%d — trace may be corrupted\n"+
					"  Fix: re-run 'glassbox debug --trace-output <file>' to regenerate a clean trace\n"+
					"  Check: if using --save-snapshots, ensure no partial writes occurred",
				i, state.Step,
			)
		}
	}

	p := &goprofile.Profile{
		SampleType: []*goprofile.ValueType{
			{Type: SampleTypeGas, Unit: SampleUnitCount},
		},
		DefaultSampleType: SampleTypeGas,
		Mapping: []*goprofile.Mapping{
			{ID: 1, Start: 0, Limit: 0, File: "soroban", HasFunctions: true},
		},
		Function: make([]*goprofile.Function, 0),
		Location: make([]*goprofile.Location, 0),
		Sample:   make([]*goprofile.Sample, 0),
	}

	funcByKey := make(map[string]*goprofile.Function)
	locByKey := make(map[string]*goprofile.Location)
	mapping := p.Mapping[0]
	var funcID, locID uint64

	nextFuncID := func() uint64 {
		funcID++
		return funcID
	}
	nextLocID := func() uint64 {
		locID++
		return locID
	}

	for i := range execTrace.States {
		state := &execTrace.States[i]
		gas := extractGasFromState(state)
		if gas < 0 {
			gas = 0
		}

		name := functionName(state)
		if name == "" {
			name = state.Operation
		}
		if name == "" {
			name = fmt.Sprintf("step_%d", state.Step)
		}

		key := name
		loc, ok := locByKey[key]
		if !ok {
			fn, ok := funcByKey[key]
			if !ok {
				fn = &goprofile.Function{
					ID:   nextFuncID(),
					Name: name,
				}
				p.Function = append(p.Function, fn)
				funcByKey[key] = fn
			}
			loc = &goprofile.Location{
				ID:      nextLocID(),
				Mapping: mapping,
				Address: uint64(state.Step),
				Line:    []goprofile.Line{{Function: fn, Line: int64(state.Step)}},
			}
			p.Location = append(p.Location, loc)
			locByKey[key] = loc
		}

		if gas > 0 {
			p.Sample = append(p.Sample, &goprofile.Sample{
				Location: []*goprofile.Location{loc},
				Value:    []int64{gas},
			})
		}
	}

	if err := p.CheckValid(); err != nil {
		return nil, fmt.Errorf("pprof profile validation failed: %w\n"+
			"  This may indicate the trace contains invalid or inconsistent data\n"+
			"  Fix: re-run the simulation to regenerate a clean trace", err)
	}

	if len(p.Sample) == 0 && len(execTrace.States) > 0 {
		// Not a hard error — zero-gas traces are valid (e.g. read-only calls).
		// Callers can check p.Sample themselves; we surface a hint via stderr
		// so the user knows why the flamegraph may be empty.
		fmt.Fprintf(os.Stderr,
			"Info: no gas samples found in trace — all steps reported zero gas usage.\n"+
				"  The pprof profile will be empty. Possible causes:\n"+
				"    - The contract does not record gas in HostState[\"gas_used\"]\n"+
				"    - The simulator version predates gas tracking\n"+
				"  Tip: use 'glassbox profile --xdr <tx.xdr>' for a live gas report\n",
		)
	}

	return p, nil
}

func functionName(state *trace.ExecutionState) string {
	if state.ContractID != "" && state.Function != "" {
		return state.ContractID + "::" + state.Function
	}
	if state.Function != "" {
		return state.Function
	}
	return ""
}

func extractGasFromState(state *trace.ExecutionState) int64 {
	if state.HostState == nil {
		return 0
	}
	v, ok := state.HostState["gas_used"]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	case uint64:
		return int64(n)
	default:
		return 0
	}
}

// WritePprof writes the trace as a pprof profile to w (gzip-compressed protobuf).
func WritePprof(execTrace *trace.ExecutionTrace, w io.Writer) error {
	p, err := TraceToPprof(execTrace)
	if err != nil {
		return err
	}
	return p.Write(w)
}
