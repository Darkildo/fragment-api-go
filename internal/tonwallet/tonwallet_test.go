package tonwallet

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Darkildo/fragment-api-go/internal/types"
)

// --- NormalizeVersion ---

func TestNormalizeVersion_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  types.WalletVersion
	}{
		{"V3R1", types.WalletV3R1},
		{"v3r1", types.WalletV3R1},
		{"V3R2", types.WalletV3R2},
		{"V4R2", types.WalletV4R2},
		{"v4r2", types.WalletV4R2},
		{"V5R1", types.WalletV5R1},
		{"W5", types.WalletV5R1}, // alias
		{"w5", types.WalletV5R1}, // case insensitive
		{"", types.WalletV4R2},   // default
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := NormalizeVersion(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("NormalizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeVersion_Invalid(t *testing.T) {
	invalids := []string{"V99", "V4R3", "abc", "V6"}
	for _, input := range invalids {
		t.Run(input, func(t *testing.T) {
			_, err := NormalizeVersion(input)
			if err == nil {
				t.Fatalf("expected error for %q", input)
			}
		})
	}
}

// --- txSem (transaction semaphore) ---

// newTestManager creates a Manager with TxSem initialised,
// without requiring a real mnemonic (for semaphore tests only).
func newTestManager() *Manager {
	return &Manager{
		TxSem: make(chan struct{}, 1),
	}
}

func TestAcquireTxLock_Immediate(t *testing.T) {
	wm := newTestManager()

	ctx := context.Background()
	if err := wm.AcquireTxLock(ctx); err != nil {
		t.Fatalf("AcquireTxLock failed: %v", err)
	}
	// Lock held — semaphore should be full.
	if len(wm.TxSem) != 1 {
		t.Errorf("TxSem len = %d, want 1", len(wm.TxSem))
	}
	wm.ReleaseTxLock()
	if len(wm.TxSem) != 0 {
		t.Errorf("TxSem len after release = %d, want 0", len(wm.TxSem))
	}
}

func TestAcquireTxLock_BlocksWhileHeld(t *testing.T) {
	wm := newTestManager()

	// Goroutine 1 acquires the lock.
	if err := wm.AcquireTxLock(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Goroutine 2 tries to acquire with a short timeout — should fail.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := wm.AcquireTxLock(ctx)
	if err == nil {
		t.Fatal("expected error: semaphore should be held")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected deadline exceeded, got: %v", err)
	}

	// Release lock, now acquire should succeed.
	wm.ReleaseTxLock()
	if err := wm.AcquireTxLock(context.Background()); err != nil {
		t.Fatalf("acquire after release failed: %v", err)
	}
	wm.ReleaseTxLock()
}

func TestAcquireTxLock_CancelledContext(t *testing.T) {
	wm := newTestManager()

	// Hold the lock.
	_ = wm.AcquireTxLock(context.Background())

	// Pre-cancelled context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := wm.AcquireTxLock(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected 'context canceled', got: %v", err)
	}

	wm.ReleaseTxLock()
}

func TestTxSem_SerializesExecution(t *testing.T) {
	wm := newTestManager()

	// Track how many goroutines are inside the critical section simultaneously.
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	const workers = 10
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx := context.Background()
			if err := wm.AcquireTxLock(ctx); err != nil {
				t.Errorf("AcquireTxLock: %v", err)
				return
			}
			defer wm.ReleaseTxLock()

			// Inside critical section.
			cur := concurrent.Add(1)
			if cur > maxConcurrent.Load() {
				maxConcurrent.Store(cur)
			}

			// Simulate work.
			time.Sleep(5 * time.Millisecond)

			concurrent.Add(-1)
		}()
	}

	wg.Wait()

	if max := maxConcurrent.Load(); max != 1 {
		t.Errorf("maxConcurrent = %d, want 1 (semaphore should serialize)", max)
	}
}

func TestTxSem_OrderIsNotGuaranteedButAllComplete(t *testing.T) {
	wm := newTestManager()

	const workers = 5
	results := make(chan int, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := wm.AcquireTxLock(ctx); err != nil {
				t.Errorf("worker %d: %v", id, err)
				return
			}
			defer wm.ReleaseTxLock()

			time.Sleep(2 * time.Millisecond)
			results <- id
		}(i)
	}

	wg.Wait()
	close(results)

	// All workers should have completed.
	got := 0
	for range results {
		got++
	}
	if got != workers {
		t.Errorf("completed %d/%d workers", got, workers)
	}
}

func TestTxSem_TimeoutWhileWaiting(t *testing.T) {
	wm := newTestManager()

	// Worker 1 holds the lock for 200ms.
	_ = wm.AcquireTxLock(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		wm.ReleaseTxLock()
	}()

	// Worker 2 has a 50ms timeout — should fail.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := wm.AcquireTxLock(ctx)
	if err == nil {
		wm.ReleaseTxLock()
		t.Fatal("expected timeout error")
	}

	// Worker 3 has a 500ms timeout — should succeed after worker 1 releases.
	ctx3, cancel3 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel3()

	if err := wm.AcquireTxLock(ctx3); err != nil {
		t.Fatalf("worker 3 should succeed after release: %v", err)
	}
	wm.ReleaseTxLock()
}

func TestTxSem_ReentrantRelease(t *testing.T) {
	wm := newTestManager()

	// Acquire and release twice in sequence — should work.
	for i := 0; i < 5; i++ {
		if err := wm.AcquireTxLock(context.Background()); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		wm.ReleaseTxLock()
	}
}

func TestNormalizeVersion_InvalidReturnsTypedError(t *testing.T) {
	err := types.NewInvalidWalletVersionError("BAD")

	if err.Version != "BAD" {
		t.Errorf("Version = %q, want BAD", err.Version)
	}
}

// --- New ---

func TestNew_Valid(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, err := New(strings.TrimSpace(mnemonic), "V4R2", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wm.Version != types.WalletV4R2 {
		t.Errorf("version = %q, want V4R2", wm.Version)
	}
	if len(wm.Mnemonic) != 24 {
		t.Errorf("mnemonic len = %d, want 24", len(wm.Mnemonic))
	}
	if wm.Testnet != false {
		t.Error("testnet should be false")
	}
}

func TestNew_Testnet(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, err := New(strings.TrimSpace(mnemonic), "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wm.Version != types.WalletV4R2 {
		t.Errorf("default version = %q, want V4R2", wm.Version)
	}
	if !wm.Testnet {
		t.Error("testnet should be true")
	}
}

func TestNew_EmptyMnemonic(t *testing.T) {
	_, err := New("", "V4R2", false)
	if err == nil {
		t.Fatal("expected error for empty mnemonic")
	}
}

func TestNew_ShortMnemonic(t *testing.T) {
	_, err := New("only three words", "V4R2", false)
	if err == nil {
		t.Fatal("expected error for short mnemonic")
	}
	if !strings.Contains(err.Error(), "24 words") {
		t.Errorf("error should mention 24 words: %v", err)
	}
}

func TestNew_InvalidVersion(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	_, err := New(strings.TrimSpace(mnemonic), "V99", false)
	if err == nil {
		t.Fatal("expected error for invalid version")
	}
}

func TestNew_DefaultVersion(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, err := New(strings.TrimSpace(mnemonic), "", false)
	if err != nil {
		t.Fatal(err)
	}
	if wm.Version != types.WalletV4R2 {
		t.Errorf("default version = %q, want V4R2", wm.Version)
	}
}

// --- Info() ---

func TestManager_Info(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, _ := New(strings.TrimSpace(mnemonic), "V5R1", false)

	info := wm.Info()
	if info.Version != types.WalletV5R1 {
		t.Errorf("Version = %q, want V5R1", info.Version)
	}
	if len(info.SupportedVersions) != len(CanonicalVersions) {
		t.Errorf("SupportedVersions len = %d, want %d", len(info.SupportedVersions), len(CanonicalVersions))
	}
	if info.Address != "" {
		t.Errorf("Address should be empty before connect, got %q", info.Address)
	}
}

func TestManager_Info_DeterministicOrder(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, _ := New(strings.TrimSpace(mnemonic), "V4R2", false)

	// Call Info() multiple times — order should always be the same.
	first := wm.Info().SupportedVersions
	for i := 0; i < 10; i++ {
		got := wm.Info().SupportedVersions
		if len(got) != len(first) {
			t.Fatalf("length changed: %d vs %d", len(got), len(first))
		}
		for j := range first {
			if got[j] != first[j] {
				t.Fatalf("order changed at index %d: %v vs %v", j, got, first)
			}
		}
	}
}
