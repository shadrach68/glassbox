// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package replay

import (
	"context"
	"fmt"

	"github.com/dotandev/glassbox/internal/errors"
)

// LedgerFetcher retrieves a ledger snapshot by sequence number.
// Implementations wrap a live RPC client or a test stub.
type LedgerFetcher interface {
	FetchLedgerSnapshot(ctx context.Context, sequence uint32) (*LedgerSnapshot, error)
}

// LedgerSnapshot is the minimal ledger state needed to resume replay.
type LedgerSnapshot struct {
	Sequence    uint32
	EntriesXDR  map[string]string
	ProtocolVer uint32
}

// ValidateLedgerSequence returns a LedgerSequenceMismatchError when
// txSequence and replaySequence differ, or nil when they match.
func ValidateLedgerSequence(txSequence, replaySequence uint32) error {
	if txSequence != replaySequence {
		return errors.WrapLedgerSequenceMismatch(txSequence, replaySequence)
	}
	return nil
}

// SequenceMismatchRecovery retries replay after fetching the exact ledger
// snapshot that the transaction references.
type SequenceMismatchRecovery struct {
	Fetcher    LedgerFetcher
	MaxRetries int
}

// NewSequenceMismatchRecovery returns a SequenceMismatchRecovery with sane defaults.
func NewSequenceMismatchRecovery(fetcher LedgerFetcher) *SequenceMismatchRecovery {
	return &SequenceMismatchRecovery{
		Fetcher:    fetcher,
		MaxRetries: 3,
	}
}

// Recover fetches the ledger snapshot for txSequence and invokes replayFn
// with the retrieved snapshot. It retries up to MaxRetries times on transient
// fetch failures.
//
// replayFn receives the fetched snapshot and should return nil on success.
func (r *SequenceMismatchRecovery) Recover(
	ctx context.Context,
	txSequence uint32,
	replayFn func(*LedgerSnapshot) error,
) error {
	var lastErr error
	for attempt := 1; attempt <= r.MaxRetries; attempt++ {
		snap, err := r.Fetcher.FetchLedgerSnapshot(ctx, txSequence)
		if err != nil {
			lastErr = err
			continue
		}
		if err := replayFn(snap); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("%w: after %d attempts: %v",
		errors.ErrLedgerSequenceMismatch, r.MaxRetries, lastErr)
}
