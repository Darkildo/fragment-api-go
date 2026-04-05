// Package tonwallet handles TON wallet operations via tonutils-go.
package tonwallet

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/Darkildo/fragment-api-go/internal/helpers"
	"github.com/Darkildo/fragment-api-go/internal/types"
)

const (
	// FragmentTxFeeNano is the fee for Fragment-initiated transactions (balance check).
	// Currently unused but reserved for future balance pre-check before Fragment purchases.
	FragmentTxFeeNano uint64 = 1_000_000 //nolint:unused // reserved for future use

	// DirectTransferFeeTON is the fee buffer for direct TON transfers.
	DirectTransferFeeTON float64 = 0.05

	// DefaultVersion is the default wallet contract version.
	DefaultVersion = types.WalletV4R2

	mainnetConfigURL = "https://ton-blockchain.github.io/global.config.json"
	testnetConfigURL = "https://ton-blockchain.github.io/testnet-global.config.json"
)

// VersionAliases maps every accepted version string (upper-case) to its
// canonical WalletVersion.
var VersionAliases = map[string]types.WalletVersion{
	"V3R1": types.WalletV3R1,
	"V3R2": types.WalletV3R2,
	"V4R2": types.WalletV4R2,
	"V5R1": types.WalletV5R1,
	"W5":   types.WalletV5R1,
}

// Manager handles TON wallet operations via tonutils-go.
//
// Transactions are serialised via a channel semaphore (TxSem) to prevent
// concurrent seqno conflicts on the TON blockchain. Only one transaction
// can be in-flight at a time. Callers waiting for the lock can cancel
// via context.
type Manager struct {
	Mnemonic []string
	Version  types.WalletVersion
	Testnet  bool

	// TxSem serialises all transaction sends (buffer size 1).
	// Write to acquire, read to release.
	TxSem chan struct{}

	// Lazily initialised on first blockchain call via sync.Once.
	Once    sync.Once
	InitErr error // sticky error from EnsureConnected
	Pool    *liteclient.ConnectionPool
	API     ton.APIClientWrapped
	Wallet  *wallet.Wallet
}

// New creates and validates a Manager.
// The actual LiteClient connection is deferred to the first blockchain call.
func New(mnemonic, version string, testnet bool) (*Manager, error) {
	if mnemonic == "" {
		return nil, types.NewWalletError("wallet mnemonic is required", nil)
	}

	ver, err := NormalizeVersion(version)
	if err != nil {
		return nil, err
	}

	words := strings.Fields(mnemonic)
	if len(words) != 24 {
		return nil, types.NewWalletError(fmt.Sprintf("mnemonic must be 24 words, got %d", len(words)), nil)
	}

	return &Manager{
		Mnemonic: words,
		Version:  ver,
		Testnet:  testnet,
		TxSem:    make(chan struct{}, 1),
	}, nil
}

// EnsureConnected lazily initialises the LiteClient pool, API client,
// and wallet instance. It is safe for concurrent use (guarded by sync.Once).
// If initialisation fails, the error is sticky — subsequent calls return
// the same error without retrying.
func (w *Manager) EnsureConnected(ctx context.Context) error {
	w.Once.Do(func() {
		w.InitErr = w.connect(ctx)
	})
	return w.InitErr
}

// connect performs the actual network initialisation. Called exactly once.
func (w *Manager) connect(ctx context.Context) error {
	cfgURL := mainnetConfigURL
	if w.Testnet {
		cfgURL = testnetConfigURL
	}

	pool := liteclient.NewConnectionPool()

	cfg, err := liteclient.GetConfigFromUrl(ctx, cfgURL)
	if err != nil {
		pool.Stop()
		return types.NewWalletError(fmt.Sprintf("fetch TON config from %s", cfgURL), err)
	}

	if err := pool.AddConnectionsFromConfig(ctx, cfg); err != nil {
		pool.Stop()
		return types.NewWalletError("connect to TON network", err)
	}

	apiClient := ton.NewAPIClient(pool, ton.ProofCheckPolicyFast).WithRetry()
	apiClient.SetTrustedBlockFromConfig(cfg)

	verCfg := w.tonutilsVersionConfig()
	wlt, err := wallet.FromSeedWithOptions(apiClient, w.Mnemonic, verCfg)
	if err != nil {
		pool.Stop()
		return types.NewWalletError("create wallet from mnemonic", err)
	}

	// All succeeded — commit state.
	w.Pool = pool
	w.API = apiClient
	w.Wallet = wlt
	return nil
}

// AcquireTxLock waits to acquire the transaction semaphore.
// Returns nil on success, or ctx.Err() if the context is cancelled/expired
// while waiting.
func (w *Manager) AcquireTxLock(ctx context.Context) error {
	select {
	case w.TxSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("waiting for transaction slot: %w", ctx.Err())
	}
}

// ReleaseTxLock releases the transaction semaphore.
func (w *Manager) ReleaseTxLock() {
	<-w.TxSem
}

// tonutilsVersionConfig returns the tonutils-go VersionConfig for the
// configured wallet version.
func (w *Manager) tonutilsVersionConfig() wallet.VersionConfig {
	var networkID int32 = wallet.MainnetGlobalID
	if w.Testnet {
		networkID = wallet.TestnetGlobalID
	}

	switch w.Version {
	case types.WalletV3R1:
		return wallet.V3R1
	case types.WalletV3R2:
		return wallet.V3R2
	case types.WalletV5R1:
		return wallet.ConfigV5R1Final{NetworkGlobalID: networkID}
	default: // WalletV4R2
		return wallet.V4R2
	}
}

// GetBalance retrieves the current wallet balance from the TON blockchain.
func (w *Manager) GetBalance(ctx context.Context) (*types.WalletBalance, error) {
	if err := w.EnsureConnected(ctx); err != nil {
		return nil, err
	}

	ctx = w.Pool.StickyContext(ctx)

	block, err := w.API.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, types.NewWalletError("get masterchain info", err)
	}

	balance, err := w.Wallet.GetBalance(ctx, block)
	if err != nil {
		return nil, types.NewWalletError("get wallet balance", err)
	}

	nanoVal := balance.Nano().Uint64()
	addr := w.Wallet.WalletAddress()

	// Determine readiness by checking the account state.
	account, err := w.API.GetAccount(ctx, block, addr)
	isReady := err == nil && account.IsActive

	return &types.WalletBalance{
		BalanceNano: nanoVal,
		BalanceTON:  float64(nanoVal) / 1e9,
		Address:     addr.String(),
		IsReady:     isReady,
		Version:     w.Version,
	}, nil
}

// SendTransaction signs and broadcasts a transaction to the TON network.
//
// Parameters:
//   - destination: TON address of the recipient.
//   - amountNano:  amount in nanotons (string).
//   - bocPayload:  base64-encoded BOC payload from Fragment API.
//
// Returns the base64-encoded transaction hash.
func (w *Manager) SendTransaction(ctx context.Context, destination, amountNano, bocPayload string) (string, error) {
	if err := w.EnsureConnected(ctx); err != nil {
		return "", err
	}

	// Validate inputs before acquiring the lock — fail fast.
	dest, err := address.ParseAddr(destination)
	if err != nil {
		return "", types.NewWalletError(fmt.Sprintf("parse destination address %q", destination), err)
	}

	nanoInt, ok := new(big.Int).SetString(amountNano, 10)
	if !ok {
		return "", types.NewWalletError(fmt.Sprintf("parse amount %q as nanotons", amountNano), nil)
	}
	amount := tlb.FromNanoTON(nanoInt)

	var body *cell.Cell
	if bocPayload != "" {
		bocBytes, err := base64.StdEncoding.DecodeString(bocPayload)
		if err != nil {
			return "", types.NewWalletError("decode BOC payload", err)
		}
		body, err = cell.FromBOC(bocBytes)
		if err != nil {
			return "", types.NewWalletError("parse BOC cell", err)
		}
	}

	// Acquire transaction lock — only one tx in-flight at a time.
	if err = w.AcquireTxLock(ctx); err != nil {
		return "", types.NewWalletError("send transaction", err)
	}
	defer w.ReleaseTxLock()

	ctx = w.Pool.StickyContext(ctx)
	msg := wallet.SimpleMessage(dest, amount, body)

	tx, _, err := w.Wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		return "", ClassifyTxError("send transaction", err)
	}

	return base64.StdEncoding.EncodeToString(tx.Hash), nil
}

// TransferTON sends TON directly to any address with an optional text memo.
func (w *Manager) TransferTON(ctx context.Context, toAddress string, amountTON float64, memo string) (*types.TransferResult, error) {
	if toAddress == "" {
		return nil, types.NewWalletError("destination address is required", nil)
	}
	if amountTON <= 0 {
		return nil, types.NewWalletError("amount must be greater than 0", nil)
	}

	if err := w.EnsureConnected(ctx); err != nil {
		return nil, err
	}

	// Validate address before acquiring the lock — fail fast.
	dest, err := address.ParseAddr(toAddress)
	if err != nil {
		return nil, types.NewWalletError(fmt.Sprintf("parse address %q", toAddress), err)
	}

	// Acquire transaction lock — only one tx in-flight at a time.
	if err = w.AcquireTxLock(ctx); err != nil {
		return nil, types.NewWalletError("transfer TON", err)
	}
	defer w.ReleaseTxLock()

	ctx = w.Pool.StickyContext(ctx)

	balBefore, err := w.GetBalance(ctx)
	if err != nil {
		return nil, err
	}

	totalRequired := amountTON + DirectTransferFeeTON
	if balBefore.BalanceTON < totalRequired {
		return nil, types.NewInsufficientBalanceError(totalRequired, balBefore.BalanceTON)
	}

	nanoInt := new(big.Int).SetUint64(uint64(helpers.RoundToNano(amountTON)))
	amount := tlb.FromNanoTON(nanoInt)

	var body *cell.Cell
	if memo != "" {
		body, err = wallet.CreateCommentCell(memo)
		if err != nil {
			return nil, types.NewWalletError("create comment cell", err)
		}
	}

	msg := wallet.SimpleMessage(dest, amount, body)

	tx, _, err := w.Wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		return nil, ClassifyTxError("transfer TON", err)
	}

	txHash := base64.StdEncoding.EncodeToString(tx.Hash)
	fromAddr := w.Wallet.WalletAddress().String()

	return &types.TransferResult{
		Success:         true,
		TransactionHash: txHash,
		FromAddress:     fromAddr,
		ToAddress:       toAddress,
		AmountTON:       amountTON,
		BalanceBefore:   balBefore.BalanceTON,
		Memo:            memo,
	}, nil
}

// ClassifyTxError wraps a transaction error into the appropriate typed error.
// If the underlying cause is [ton.ErrTxWasNotConfirmed], it returns a
// [types.TransactionNotConfirmedError]; otherwise a generic [types.TransactionError].
func ClassifyTxError(msg string, err error) error {
	if errors.Is(err, ton.ErrTxWasNotConfirmed) {
		return types.NewTransactionNotConfirmedError(err)
	}
	return types.NewTransactionError(msg, err)
}

// CanonicalVersions is a fixed-order list of canonical (non-alias) wallet versions.
var CanonicalVersions = []types.WalletVersion{
	types.WalletV4R2, types.WalletV5R1, types.WalletV3R2, types.WalletV3R1,
}

// Info returns wallet metadata as a typed struct.
func (w *Manager) Info() types.WalletInfo {
	wi := types.WalletInfo{
		Version:           w.Version,
		SupportedVersions: CanonicalVersions,
	}
	if w.Wallet != nil {
		wi.Address = w.Wallet.WalletAddress().String()
	}
	return wi
}

// NormalizeVersion validates and normalises a version string (case-insensitive,
// resolves aliases like "w5" -> WalletV5R1).
// Empty string defaults to WalletV4R2.
func NormalizeVersion(v string) (types.WalletVersion, error) {
	if v == "" {
		return DefaultVersion, nil
	}
	canonical, ok := VersionAliases[strings.ToUpper(v)]
	if !ok {
		return "", types.NewInvalidWalletVersionError(v)
	}
	return canonical, nil
}
