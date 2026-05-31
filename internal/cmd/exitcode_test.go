// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"testing"

	"github.com/dotandev/glassbox/internal/errors"
)

func TestExitCodeFor(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil error is success", nil, ExitSuccess},
		{"interrupt is 130", ErrInterrupted, InterruptExitCode},
		{"validation error is user error", errors.WrapValidationError("bad input"), ExitUserError},
		{"simulation logic error is user error", errors.WrapSimulationLogicError("bad logic"), ExitUserError},
		{"config error is config error", errors.WrapConfigError("bad config", fmt.Errorf("missing")), ExitConfigError},
		{"unknown error is internal", fmt.Errorf("unknown failure"), ExitInternalError},
		{"sentinel ErrValidationFailed is user error", errors.ErrValidationFailed, ExitUserError},
		{"sentinel ErrConfigFailed is config error", errors.ErrConfigFailed, ExitConfigError},
		{"sentinel ErrSimulatorNotFound is config error", errors.ErrSimulatorNotFound, ExitConfigError},
		{"sentinel ErrTransactionNotFound is user error", errors.ErrTransactionNotFound, ExitUserError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExitCodeFor(tt.err)
			if got != tt.want {
				t.Errorf("ExitCodeFor(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}
