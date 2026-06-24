// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Package perfmetrics provides lightweight timing metrics for debug sessions.
// It captures RPC call durations and simulator execution time, then renders a
// human-readable summary that is printed at the end of a debug session when
// the --show-metrics flag is set.
package perfmetrics

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"
)

// SlowRPCThreshold is the duration above which a single RPC call is flagged as
// slow in the performance summary. Callers may override this for testing.
var SlowRPCThreshold = 3 * time.Second

// RPCRecord holds timing data for a single RPC call.
type RPCRecord struct {
	Method   string
	Duration time.Duration
	Err      bool
}

// Collector accumulates timing metrics during a debug session.
type Collector struct {
	mu          sync.Mutex
	rpcRecords  []RPCRecord
	simStart    time.Time
	simDuration time.Duration
	simRecorded bool
}

// NewCollector creates a ready-to-use Collector.
func NewCollector() *Collector {
	return &Collector{}
}

// RecordRPC records the duration of a single RPC call.
func (c *Collector) RecordRPC(method string, d time.Duration, err bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rpcRecords = append(c.rpcRecords, RPCRecord{Method: method, Duration: d, Err: err})
}

// StartSim marks the start of simulator execution.
func (c *Collector) StartSim() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.simStart = time.Now()
}

// StopSim marks the end of simulator execution.
func (c *Collector) StopSim() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.simStart.IsZero() {
		c.simDuration = time.Since(c.simStart)
		c.simRecorded = true
	}
}

// MethodSummary holds per-method aggregated statistics.
type MethodSummary struct {
	Method string
	Calls  int
	Errors int
	Total  time.Duration
	Min    time.Duration
	Max    time.Duration
}

// Avg returns the average call duration for this method.
func (m MethodSummary) Avg() time.Duration {
	if m.Calls == 0 {
		return 0
	}
	return m.Total / time.Duration(m.Calls)
}

// Summary holds the aggregated metrics ready for display.
type Summary struct {
	RPCCalls    int
	RPCErrors   int
	RPCTotal    time.Duration
	RPCMin      time.Duration
	RPCMax      time.Duration
	SimDuration time.Duration
	SimRecorded bool

	// ByMethod groups per-method breakdown when multiple RPC methods were used.
	ByMethod []MethodSummary

	// SlowCalls lists RPC calls that exceeded SlowRPCThreshold.
	SlowCalls []RPCRecord
}

// Summarize computes aggregate statistics from the collected records.
func (c *Collector) Summarize() Summary {
	c.mu.Lock()
	defer c.mu.Unlock()

	s := Summary{
		SimDuration: c.simDuration,
		SimRecorded: c.simRecorded,
	}

	methodMap := make(map[string]*MethodSummary)

	for _, r := range c.rpcRecords {
		s.RPCCalls++
		s.RPCTotal += r.Duration
		if r.Err {
			s.RPCErrors++
		}
		if s.RPCMin == 0 || r.Duration < s.RPCMin {
			s.RPCMin = r.Duration
		}
		if r.Duration > s.RPCMax {
			s.RPCMax = r.Duration
		}
		if r.Duration >= SlowRPCThreshold {
			s.SlowCalls = append(s.SlowCalls, r)
		}

		m, ok := methodMap[r.Method]
		if !ok {
			m = &MethodSummary{Method: r.Method}
			methodMap[r.Method] = m
		}
		m.Calls++
		m.Total += r.Duration
		if r.Err {
			m.Errors++
		}
		if m.Min == 0 || r.Duration < m.Min {
			m.Min = r.Duration
		}
		if r.Duration > m.Max {
			m.Max = r.Duration
		}
	}

	// Only populate ByMethod when more than one distinct RPC method was called —
	// otherwise the aggregate totals above are sufficient.
	if len(methodMap) > 1 {
		for _, m := range methodMap {
			s.ByMethod = append(s.ByMethod, *m)
		}
		sort.Slice(s.ByMethod, func(i, j int) bool {
			return s.ByMethod[i].Total > s.ByMethod[j].Total
		})
	}

	return s
}

// Print writes the human-readable performance summary to w (defaults to os.Stdout).
func (c *Collector) Print(w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	s := c.Summarize()

	fmt.Fprintln(w, "\n── Performance Summary ──────────────────────────────")
	fmt.Fprintf(w, "  RPC calls     : %d", s.RPCCalls)
	if s.RPCErrors > 0 {
		fmt.Fprintf(w, " (%d error(s))", s.RPCErrors)
	}
	fmt.Fprintln(w)

	if s.RPCCalls > 0 {
		fmt.Fprintf(w, "  RPC total     : %s\n", s.RPCTotal.Round(time.Millisecond))
		fmt.Fprintf(w, "  RPC min/max   : %s / %s\n",
			s.RPCMin.Round(time.Millisecond),
			s.RPCMax.Round(time.Millisecond))
		avg := s.RPCTotal / time.Duration(s.RPCCalls)
		fmt.Fprintf(w, "  RPC avg       : %s\n", avg.Round(time.Millisecond))

		// Per-method breakdown (only when multiple methods were recorded).
		if len(s.ByMethod) > 0 {
			fmt.Fprintln(w, "\n  Per-method breakdown:")
			for _, m := range s.ByMethod {
				line := fmt.Sprintf("    %-28s calls=%-4d total=%-10s avg=%s",
					m.Method, m.Calls,
					m.Total.Round(time.Millisecond),
					m.Avg().Round(time.Millisecond),
				)
				if m.Errors > 0 {
					line += fmt.Sprintf("  errors=%d", m.Errors)
				}
				fmt.Fprintln(w, line)
			}
		}

		// Slow-call warnings.
		if len(s.SlowCalls) > 0 {
			fmt.Fprintf(w, "\n  ⚠  Slow RPC calls (>%s):\n", SlowRPCThreshold)
			for _, r := range s.SlowCalls {
				suffix := ""
				if r.Err {
					suffix = " [error]"
				}
				fmt.Fprintf(w, "     %-28s %s%s\n",
					r.Method,
					r.Duration.Round(time.Millisecond),
					suffix,
				)
			}
			fmt.Fprintln(w, "  Tip: consider using --rpc-url to switch to a faster RPC endpoint,")
			fmt.Fprintln(w, "       or check your network connection.")
		}
	}

	if s.SimRecorded {
		fmt.Fprintf(w, "  Replay time   : %s\n", s.SimDuration.Round(time.Millisecond))
	}
	fmt.Fprintln(w, "─────────────────────────────────────────────────────")
}

// PrintJSON writes the performance summary as a JSON object to w.
// It is used when the caller requests machine-readable output (e.g. --format json).
func (c *Collector) PrintJSON(w io.Writer) error {
	if w == nil {
		w = os.Stdout
	}
	s := c.Summarize()

	type methodJSON struct {
		Method     string  `json:"method"`
		Calls      int     `json:"calls"`
		Errors     int     `json:"errors,omitempty"`
		TotalMs    float64 `json:"total_ms"`
		AvgMs      float64 `json:"avg_ms"`
		MinMs      float64 `json:"min_ms"`
		MaxMs      float64 `json:"max_ms"`
	}
	type slowJSON struct {
		Method    string  `json:"method"`
		DurationMs float64 `json:"duration_ms"`
		Error     bool    `json:"error,omitempty"`
	}
	type output struct {
		RPCCalls    int          `json:"rpc_calls"`
		RPCErrors   int          `json:"rpc_errors,omitempty"`
		RPCTotalMs  float64      `json:"rpc_total_ms"`
		RPCMinMs    float64      `json:"rpc_min_ms,omitempty"`
		RPCMaxMs    float64      `json:"rpc_max_ms,omitempty"`
		RPCAvgMs    float64      `json:"rpc_avg_ms,omitempty"`
		SimMs       float64      `json:"sim_ms,omitempty"`
		ByMethod    []methodJSON `json:"by_method,omitempty"`
		SlowCalls   []slowJSON   `json:"slow_calls,omitempty"`
	}

	ms := func(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

	out := output{
		RPCCalls:   s.RPCCalls,
		RPCErrors:  s.RPCErrors,
		RPCTotalMs: ms(s.RPCTotal),
	}
	if s.RPCCalls > 0 {
		out.RPCMinMs = ms(s.RPCMin)
		out.RPCMaxMs = ms(s.RPCMax)
		out.RPCAvgMs = ms(s.RPCTotal / time.Duration(s.RPCCalls))
	}
	if s.SimRecorded {
		out.SimMs = ms(s.SimDuration)
	}
	for _, m := range s.ByMethod {
		out.ByMethod = append(out.ByMethod, methodJSON{
			Method:  m.Method,
			Calls:   m.Calls,
			Errors:  m.Errors,
			TotalMs: ms(m.Total),
			AvgMs:   ms(m.Avg()),
			MinMs:   ms(m.Min),
			MaxMs:   ms(m.Max),
		})
	}
	for _, r := range s.SlowCalls {
		out.SlowCalls = append(out.SlowCalls, slowJSON{
			Method:     r.Method,
			DurationMs: ms(r.Duration),
			Error:      r.Err,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
