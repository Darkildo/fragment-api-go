package fragment

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

const (
	transferFeeNano uint64  = 1_000_000 // 0.001 TON
	transferFeeTON  float64 = 0.05      // fee buffer for direct transfers
	defaultVersion          = WalletV4R2

	mainnetConfigURL = "https://ton-blockchain.github.io/global.config.json"
	testnetConfigURL = "https://ton-blockchain.github.io/testnet-global.config.json"
)

// versionAliases maps every accepted version string (upper-case) to its
// canonical WalletVersion.
var versionAliases = map[string]WalletVersion{
	"V3R1": WalletV3R1,
	"V3R2": WalletV3R2,
	"V4R2": WalletV4R2,
	"V5R1": WalletV5R1,
	"W5":   WalletV5R1,
}

// walletManager handles TON wallet operations via tonutils-go.
type walletManager struct {
	mnemonic []string
	version  WalletVersion
	testnet  bool

	// Lazily initialised on first blockchain call.
	pool   *liteclient.ConnectionPool
	api    ton.APIClientWrapped
	wallet *wallet.Wallet
}

// newWalletManager creates and validates a walletManager.
// The actual LiteClient connection is deferred to the first blockchain call.
func newWalletManager(mnemonic, version string, testnet bool) (*walletManager, error) {
	if mnemonic == "" {
		return nil, newWalletError("wallet mnemonic is required", nil)
	}

	ver, err := normalizeVersion(version)
	if err != nil {
		return nil, err
	}

	words := strings.Fields(mnemonic)
	if len(words) != 24 {
		return nil, newWalletError(fmt.Sprintf("mnemonic must be 24 words, got %d", len(words)), nil)
	}

	return &walletManager{
		mnemonic: words,
		version:  ver,
		testnet:  testnet,
	}, nil
}

// ensureConnected lazily initialises the LiteClient pool, API client,
// and wallet instance on first call. Subsequent calls are no-ops.
func (w *walletManager) ensureConnected(ctx context.Context) error {
	if w.wallet != nil {
		return nil
	}

	cfgURL := mainnetConfigURL
	if w.testnet {
		cfgURL = testnetConfigURL
	}

	w.pool = liteclient.NewConnectionPool()

	cfg, err := liteclient.GetConfigFromUrl(ctx, cfgURL)
	if err != nil {
		return newWalletError(fmt.Sprintf("fetch TON config from %s", cfgURL), err)
	}

	if err := w.pool.AddConnectionsFromConfig(ctx, cfg); err != nil {
		return newWalletError("connect to TON network", err)
	}

	apiClient := ton.NewAPIClient(w.pool, ton.ProofCheckPolicyFast).WithRetry()
	apiClient.SetTrustedBlockFromConfig(cfg)
	w.api = apiClient

	verCfg := w.tonutilsVersionConfig()
	wlt, err := wallet.FromSeedWithOptions(apiClient, w.mnemonic, verCfg)
	if err != nil {
		return newWalletError("create wallet from mnemonic", err)
	}
	w.wallet = wlt

	return nil
}

// tonutilsVersionConfig returns the tonutils-go VersionConfig for the
// configured wallet version.
func (w *walletManager) tonutilsVersionConfig() wallet.VersionConfig {
	var networkID int32 = wallet.MainnetGlobalID
	if w.testnet {
		networkID = wallet.TestnetGlobalID
	}

	switch w.version {
	case WalletV3R1:
		return wallet.V3R1
	case WalletV3R2:
		return wallet.V3R2
	case WalletV5R1:
		return wallet.ConfigV5R1Final{NetworkGlobalID: networkID}
	default: // WalletV4R2
		return wallet.V4R2
	}
}

// getBalance retrieves the current wallet balance from the TON blockchain.
func (w *walletManager) getBalance(ctx context.Context) (*WalletBalance, error) {
	if err := w.ensureConnected(ctx); err != nil {
		return nil, err
	}

	ctx = w.pool.StickyContext(ctx)

	block, err := w.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, newWalletError("get masterchain info", err)
	}

	balance, err := w.wallet.GetBalance(ctx, block)
	if err != nil {
		return nil, newWalletError("get wallet balance", err)
	}

	nanoVal := balance.Nano().Uint64()
	addr := w.wallet.WalletAddress()

	// Determine readiness by checking the account state.
	account, err := w.api.GetAccount(ctx, block, addr)
	isReady := err == nil && account.IsActive

	return &WalletBalance{
		BalanceNano: nanoVal,
		BalanceTON:  float64(nanoVal) / 1e9,
		Address:     addr.String(),
		IsReady:     isReady,
		Version:     w.version,
	}, nil
}

// sendTransaction signs and broadcasts a transaction to the TON network.
//
// Parameters:
//   - destination: TON address of the recipient.
//   - amountNano:  amount in nanotons (string).
//   - bocPayload:  base64-encoded BOC payload from Fragment API.
//
// Returns the base64-encoded transaction hash.
func (w *walletManager) sendTransaction(ctx context.Context, destination, amountNano, bocPayload string) (string, error) {
	if err := w.ensureConnected(ctx); err != nil {
		return "", err
	}

	ctx = w.pool.StickyContext(ctx)

	dest, err := address.ParseAddr(destination)
	if err != nil {
		return "", newWalletError(fmt.Sprintf("parse destination address %q", destination), err)
	}

	nanoInt, ok := new(big.Int).SetString(amountNano, 10)
	if !ok {
		return "", newWalletError(fmt.Sprintf("parse amount %q as nanotons", amountNano), nil)
	}
	amount := tlb.FromNanoTON(nanoInt)

	var body *cell.Cell
	if bocPayload != "" {
		bocBytes, err := base64.StdEncoding.DecodeString(bocPayload)
		if err != nil {
			return "", newWalletError("decode BOC payload", err)
		}
		body, err = cell.FromBOC(bocBytes)
		if err != nil {
			return "", newWalletError("parse BOC cell", err)
		}
	}

	msg := wallet.SimpleMessage(dest, amount, body)

	tx, _, err := w.wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		return "", newTransactionError("send transaction", err)
	}

	return base64.StdEncoding.EncodeToString(tx.Hash), nil
}

// transferTON sends TON directly to any address with an optional text memo.
func (w *walletManager) transferTON(ctx context.Context, toAddress string, amountTON float64, memo string) (*TransferResult, error) {
	if toAddress == "" {
		return nil, newWalletError("destination address is required", nil)
	}
	if amountTON <= 0 {
		return nil, newWalletError("amount must be greater than 0", nil)
	}

	if err := w.ensureConnected(ctx); err != nil {
		return nil, err
	}

	ctx = w.pool.StickyContext(ctx)

	dest, err := address.ParseAddr(toAddress)
	if err != nil {
		return nil, newWalletError(fmt.Sprintf("parse address %q", toAddress), err)
	}

	balBefore, err := w.getBalance(ctx)
	if err != nil {
		return nil, err
	}

	totalRequired := amountTON + transferFeeTON
	if balBefore.BalanceTON < totalRequired {
		return nil, newInsufficientBalanceError(totalRequired, balBefore.BalanceTON)
	}

	nanoInt := new(big.Int).SetUint64(uint64(roundToNano(amountTON)))
	amount := tlb.FromNanoTON(nanoInt)

	var body *cell.Cell
	if memo != "" {
		body, err = wallet.CreateCommentCell(memo)
		if err != nil {
			return nil, newWalletError("create comment cell", err)
		}
	}

	msg := wallet.SimpleMessage(dest, amount, body)

	tx, _, err := w.wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		return nil, newTransactionError("transfer TON", err)
	}

	txHash := base64.StdEncoding.EncodeToString(tx.Hash)
	fromAddr := w.wallet.WalletAddress().String()

	return &TransferResult{
		Success:         true,
		TransactionHash: txHash,
		FromAddress:     fromAddr,
		ToAddress:       toAddress,
		AmountTON:       amountTON,
		BalanceBefore:   balBefore.BalanceTON,
		Memo:            memo,
	}, nil
}

// info returns wallet metadata as a typed struct.
func (w *walletManager) info() WalletInfo {
	versions := make([]WalletVersion, 0, len(versionAliases))
	seen := map[WalletVersion]bool{}
	for _, v := range versionAliases {
		if !seen[v] {
			versions = append(versions, v)
			seen[v] = true
		}
	}

	wi := WalletInfo{
		Version:           w.version,
		SupportedVersions: versions,
	}
	if w.wallet != nil {
		wi.Address = w.wallet.WalletAddress().String()
	}
	return wi
}

// normalizeVersion validates and normalises a version string (case-insensitive,
// resolves aliases like "w5" -> WalletV5R1).
// Empty string defaults to WalletV4R2.
func normalizeVersion(v string) (WalletVersion, error) {
	if v == "" {
		return defaultVersion, nil
	}
	canonical, ok := versionAliases[strings.ToUpper(v)]
	if !ok {
		return "", newInvalidWalletVersionError(v)
	}
	return canonical, nil
}
