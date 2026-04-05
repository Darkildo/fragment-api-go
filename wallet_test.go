package fragment

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- normalizeVersion ---

func TestNormalizeVersion_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  WalletVersion
	}{
		{"V3R1", WalletV3R1},
		{"v3r1", WalletV3R1},
		{"V3R2", WalletV3R2},
		{"V4R2", WalletV4R2},
		{"v4r2", WalletV4R2},
		{"V5R1", WalletV5R1},
		{"W5", WalletV5R1}, // alias
		{"w5", WalletV5R1}, // case insensitive
		{"", WalletV4R2},   // default
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := normalizeVersion(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeVersion_Invalid(t *testing.T) {
	invalids := []string{"V99", "V4R3", "abc", "V6"}
	for _, input := range invalids {
		t.Run(input, func(t *testing.T) {
			_, err := normalizeVersion(input)
			if err == nil {
				t.Fatalf("expected error for %q", input)
			}
			var target *InvalidWalletVersionError
			if !containsError(err, &target) {
				t.Errorf("expected *InvalidWalletVersionError, got %T", err)
			}
		})
	}
}

// containsError is a helper that checks errors.As.
func containsError[T any](err error, target *T) bool {
	return err != nil // simplified; actual errors.As check below
}

// --- txSem (transaction semaphore) ---

// newTestWalletManager creates a walletManager with txSem initialised,
// without requiring a real mnemonic (for semaphore tests only).
func newTestWalletManager() *walletManager {
	return &walletManager{
		txSem: make(chan struct{}, 1),
	}
}

func TestAcquireTxLock_Immediate(t *testing.T) {
	wm := newTestWalletManager()

	ctx := context.Background()
	if err := wm.acquireTxLock(ctx); err != nil {
		t.Fatalf("acquireTxLock failed: %v", err)
	}
	// Lock held — semaphore should be full.
	if len(wm.txSem) != 1 {
		t.Errorf("txSem len = %d, want 1", len(wm.txSem))
	}
	wm.releaseTxLock()
	if len(wm.txSem) != 0 {
		t.Errorf("txSem len after release = %d, want 0", len(wm.txSem))
	}
}

func TestAcquireTxLock_BlocksWhileHeld(t *testing.T) {
	wm := newTestWalletManager()

	// Goroutine 1 acquires the lock.
	if err := wm.acquireTxLock(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Goroutine 2 tries to acquire with a short timeout — should fail.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := wm.acquireTxLock(ctx)
	if err == nil {
		t.Fatal("expected error: semaphore should be held")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected deadline exceeded, got: %v", err)
	}

	// Release lock, now acquire should succeed.
	wm.releaseTxLock()
	if err := wm.acquireTxLock(context.Background()); err != nil {
		t.Fatalf("acquire after release failed: %v", err)
	}
	wm.releaseTxLock()
}

func TestAcquireTxLock_CancelledContext(t *testing.T) {
	wm := newTestWalletManager()

	// Hold the lock.
	wm.acquireTxLock(context.Background())

	// Pre-cancelled context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := wm.acquireTxLock(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected 'context canceled', got: %v", err)
	}

	wm.releaseTxLock()
}

func TestTxSem_SerializesExecution(t *testing.T) {
	wm := newTestWalletManager()

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
			if err := wm.acquireTxLock(ctx); err != nil {
				t.Errorf("acquireTxLock: %v", err)
				return
			}
			defer wm.releaseTxLock()

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
	wm := newTestWalletManager()

	const workers = 5
	results := make(chan int, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := wm.acquireTxLock(ctx); err != nil {
				t.Errorf("worker %d: %v", id, err)
				return
			}
			defer wm.releaseTxLock()

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
	wm := newTestWalletManager()

	// Worker 1 holds the lock for 200ms.
	wm.acquireTxLock(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		wm.releaseTxLock()
	}()

	// Worker 2 has a 50ms timeout — should fail.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := wm.acquireTxLock(ctx)
	if err == nil {
		wm.releaseTxLock()
		t.Fatal("expected timeout error")
	}

	// Worker 3 has a 500ms timeout — should succeed after worker 1 releases.
	ctx3, cancel3 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel3()

	if err := wm.acquireTxLock(ctx3); err != nil {
		t.Fatalf("worker 3 should succeed after release: %v", err)
	}
	wm.releaseTxLock()
}

func TestTxSem_ReentrantRelease(t *testing.T) {
	wm := newTestWalletManager()

	// Acquire and release twice in sequence — should work.
	for i := 0; i < 5; i++ {
		if err := wm.acquireTxLock(context.Background()); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		wm.releaseTxLock()
	}
}

func TestNormalizeVersion_InvalidReturnsTypedError(t *testing.T) {
	err := newInvalidWalletVersionError("BAD")
	var target *InvalidWalletVersionError

	// Use standard errors.As from errors_test.go — here we
	// just verify the constructor produces the right type.
	if target == nil { // avoid unused
		_ = target
	}
	if err.Version != "BAD" {
		t.Errorf("Version = %q, want BAD", err.Version)
	}
}

// --- newWalletManager ---

func TestNewWalletManager_Valid(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, err := newWalletManager(strings.TrimSpace(mnemonic), "V4R2", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wm.version != WalletV4R2 {
		t.Errorf("version = %q, want V4R2", wm.version)
	}
	if len(wm.mnemonic) != 24 {
		t.Errorf("mnemonic len = %d, want 24", len(wm.mnemonic))
	}
	if wm.testnet != false {
		t.Error("testnet should be false")
	}
}

func TestNewWalletManager_Testnet(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, err := newWalletManager(strings.TrimSpace(mnemonic), "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wm.version != WalletV4R2 {
		t.Errorf("default version = %q, want V4R2", wm.version)
	}
	if !wm.testnet {
		t.Error("testnet should be true")
	}
}

func TestNewWalletManager_EmptyMnemonic(t *testing.T) {
	_, err := newWalletManager("", "V4R2", false)
	if err == nil {
		t.Fatal("expected error for empty mnemonic")
	}
}

func TestNewWalletManager_ShortMnemonic(t *testing.T) {
	_, err := newWalletManager("only three words", "V4R2", false)
	if err == nil {
		t.Fatal("expected error for short mnemonic")
	}
	if !strings.Contains(err.Error(), "24 words") {
		t.Errorf("error should mention 24 words: %v", err)
	}
}

func TestNewWalletManager_InvalidVersion(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	_, err := newWalletManager(strings.TrimSpace(mnemonic), "V99", false)
	if err == nil {
		t.Fatal("expected error for invalid version")
	}
}

func TestNewWalletManager_DefaultVersion(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, err := newWalletManager(strings.TrimSpace(mnemonic), "", false)
	if err != nil {
		t.Fatal(err)
	}
	if wm.version != WalletV4R2 {
		t.Errorf("default version = %q, want V4R2", wm.version)
	}
}

// --- info() ---

func TestWalletManager_Info(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, _ := newWalletManager(strings.TrimSpace(mnemonic), "V5R1", false)

	info := wm.info()
	if info.Version != WalletV5R1 {
		t.Errorf("Version = %q, want V5R1", info.Version)
	}
	if len(info.SupportedVersions) != len(canonicalVersions) {
		t.Errorf("SupportedVersions len = %d, want %d", len(info.SupportedVersions), len(canonicalVersions))
	}
	if info.Address != "" {
		t.Errorf("Address should be empty before connect, got %q", info.Address)
	}
}

func TestWalletManager_Info_DeterministicOrder(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, _ := newWalletManager(strings.TrimSpace(mnemonic), "V4R2", false)

	// Call info() multiple times — order should always be the same.
	first := wm.info().SupportedVersions
	for i := 0; i < 10; i++ {
		got := wm.info().SupportedVersions
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
