package marketdata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"investplanner/server/internal/database"
	"investplanner/server/internal/repository"
)

type SnapshotStore interface {
	SaveSnapshot(database.MarketDataSnapshot) error
	GetSnapshot(string, string) (database.MarketDataSnapshot, error)
}
type SnapshotService struct {
	Store    SnapshotStore
	Now      func() time.Time
	FreshFor map[string]time.Duration
}

func (s *SnapshotService) Put(dataType, entityKey string, value any, meta SourceMeta) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	return s.Store.SaveSnapshot(database.MarketDataSnapshot{DataType: dataType, EntityKey: entityKey, Provider: meta.Provider, SourceURL: meta.SourceURL, ObservedAt: meta.ObservedAt, FetchedAt: meta.FetchedAt, Status: string(Fresh), RawHash: hex.EncodeToString(sum[:]), NormalizedJSON: raw})
}
func (s *SnapshotService) Get(dataType, entityKey string, destination any) (Availability, error) {
	snapshot, err := s.Store.GetSnapshot(dataType, entityKey)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return Unavailable, nil
		}
		return Unavailable, err
	}
	if err := json.Unmarshal(snapshot.NormalizedJSON, destination); err != nil {
		return Unavailable, fmt.Errorf("invalid stored snapshot")
	}
	freshFor := s.FreshFor[dataType]
	if freshFor <= 0 {
		freshFor = 24 * time.Hour
	}
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	return Classify(snapshot.FetchedAt, now, freshFor), nil
}

type RefreshJob struct {
	DataType, EntityKey string
	Fetch               func(context.Context) (any, SourceMeta, error)
}
type breaker struct {
	failures  int
	openUntil time.Time
}
type Refresher struct {
	Snapshots    *SnapshotService
	Timeout      time.Duration
	Retries      int
	sem          chan struct{}
	mu           sync.Mutex
	breakers     map[string]breaker
	FailureLimit int
	Cooldown     time.Duration
}

func NewRefresher(s *SnapshotService, concurrency int) *Refresher {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Refresher{Snapshots: s, Timeout: 5 * time.Second, Retries: 2, sem: make(chan struct{}, concurrency), breakers: map[string]breaker{}, FailureLimit: 3, Cooldown: time.Minute}
}
func (r *Refresher) Run(ctx context.Context, job RefreshJob) error {
	r.mu.Lock()
	state := r.breakers[job.DataType]
	if time.Now().Before(state.openUntil) {
		r.mu.Unlock()
		return fmt.Errorf("circuit_open")
	}
	r.mu.Unlock()
	select {
	case r.sem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-r.sem }()
	var last error
	for attempt := 0; attempt <= r.Retries; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, r.Timeout)
		value, meta, err := job.Fetch(callCtx)
		cancel()
		if err == nil {
			if saveErr := r.Snapshots.Put(job.DataType, job.EntityKey, value, meta); saveErr != nil {
				return saveErr
			}
			r.mu.Lock()
			delete(r.breakers, job.DataType)
			r.mu.Unlock()
			return nil
		}
		last = err
	}
	r.mu.Lock()
	state = r.breakers[job.DataType]
	state.failures++
	if state.failures >= r.FailureLimit {
		state.openUntil = time.Now().Add(r.Cooldown)
	}
	r.breakers[job.DataType] = state
	r.mu.Unlock()
	return fmt.Errorf("refresh_failed: %w", last)
}

// CrossValidate accepts a fresh primary value only when an optional secondary
// agrees within tolerance. A mismatch preserves the last successful snapshot.
func CrossValidate(primary, secondary *float64, tolerance float64) (*float64, string, error) {
	if primary == nil {
		return nil, "snapshot", fmt.Errorf("primary_unavailable")
	}
	if secondary == nil {
		return primary, "primary", nil
	}
	difference := *primary - *secondary
	if difference < 0 {
		difference = -difference
	}
	if difference > tolerance {
		return nil, "snapshot", fmt.Errorf("source_mismatch")
	}
	return primary, "cross_validated", nil
}
