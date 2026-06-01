// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package replay

import (
	"context"
	"errors"
	"fmt"
	"testing"

	glassErrors "github.com/dotandev/glassbox/internal/errors"
)

// --- ValidateLedgerSequence ---

func TestValidateLedgerSequence_Match(t *testing.T) {
	if err := ValidateLedgerSequence(100, 100); err != nil {
		t.Errorf("expected nil for matching sequences, got %v", err)
	}
}

func TestValidateLedgerSequence_Mismatch(t *testing.T) {
	err := ValidateLedgerSequence(200, 100)
	if err == nil {
		t.Fatal("expected error for mismatched sequences")
	}
	if !errors.Is(err, glassErrors.ErrLedgerSequenceMismatch) {
		t.Errorf("expected ErrLedgerSequenceMismatch, got %T: %v", err, err)
	}

	var mismatch *glassErrors.LedgerSequenceMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("expected *LedgerSequenceMismatchError, got %T", err)
	}
	if mismatch.TxSequence != 200 {
		t.Errorf("expected TxSequence 200, got %d", mismatch.TxSequence)
	}
	if mismatch.ReplaySequence != 100 {
		t.Errorf("expected ReplaySequence 100, got %d", mismatch.ReplaySequence)
	}
}

func TestValidateLedgerSequence_ZeroMatch(t *testing.T) {
	if err := ValidateLedgerSequence(0, 0); err != nil {
		t.Errorf("expected nil for zero sequences, got %v", err)
	}
}

// --- SequenceMismatchRecovery ---

type stubFetcher struct {
	snap *LedgerSnapshot
	err  error
	calls int
}

func (s *stubFetcher) FetchLedgerSnapshot(_ context.Context, seq uint32) (*LedgerSnapshot, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	if s.snap != nil {
		s.snap.Sequence = seq
	}
	return s.snap, nil
}

func TestRecover_Success(t *testing.T) {
	fetcher := &stubFetcher{
		snap: &LedgerSnapshot{EntriesXDR: map[string]string{"k": "v"}, ProtocolVer: 21},
	}
	rec := NewSequenceMismatchRecovery(fetcher)

	var received *LedgerSnapshot
	err := rec.Recover(context.Background(), 42, func(s *LedgerSnapshot) error {
		received = s
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received == nil {
		t.Fatal("expected snapshot to be passed to replayFn")
	}
	if received.Sequence != 42 {
		t.Errorf("expected sequence 42, got %d", received.Sequence)
	}
}

func TestRecover_FetchError_Retries(t *testing.T) {
	fetcher := &stubFetcher{err: fmt.Errorf("transient network error")}
	rec := NewSequenceMismatchRecovery(fetcher)
	rec.MaxRetries = 3

	err := rec.Recover(context.Background(), 10, func(_ *LedgerSnapshot) error { return nil })
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if fetcher.calls != 3 {
		t.Errorf("expected 3 fetch attempts, got %d", fetcher.calls)
	}
	if !errors.Is(err, glassErrors.ErrLedgerSequenceMismatch) {
		t.Errorf("expected ErrLedgerSequenceMismatch in wrapped error, got: %v", err)
	}
}

func TestRecover_ReplayFnError(t *testing.T) {
	fetcher := &stubFetcher{
		snap: &LedgerSnapshot{},
	}
	rec := NewSequenceMismatchRecovery(fetcher)
	replayErr := fmt.Errorf("replay failed")

	err := rec.Recover(context.Background(), 5, func(_ *LedgerSnapshot) error {
		return replayErr
	})
	if err == nil {
		t.Fatal("expected error from replayFn to be propagated")
	}
	if !errors.Is(err, replayErr) {
		t.Errorf("expected replayErr to be wrapped, got: %v", err)
	}
}

func TestRecover_SuccessAfterRetry(t *testing.T) {
	callCount := 0
	fetcher := &stubFetcher{}
	rec := NewSequenceMismatchRecovery(fetcher)
	rec.MaxRetries = 3

	err := rec.Recover(context.Background(), 7, func(_ *LedgerSnapshot) error {
		callCount++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected replayFn called once, got %d", callCount)
	}
}

func TestNewSequenceMismatchRecovery_Defaults(t *testing.T) {
	fetcher := &stubFetcher{}
	rec := NewSequenceMismatchRecovery(fetcher)
	if rec.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", rec.MaxRetries)
	}
	if rec.Fetcher == nil {
		t.Error("expected non-nil Fetcher")
	}
}
