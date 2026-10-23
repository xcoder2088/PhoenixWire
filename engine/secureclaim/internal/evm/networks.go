package evm

import (
	"errors"
	"math/big"
	"os"
	"strings"
)

// Asset IDs are the public names of the supported stablecoins: USDT on Ethereum, USDC and USDT on Polygon.
const (
	AssetUSDTEthereum = "usdt_ethereum"
	AssetUSDCPolygon  = "usdc_polygon"
	AssetUSDTPolygon  = "usdt_polygon"
)

type Network struct {
	Asset         string
	Chain         string // ethereum | polygon (what the user sees)
	ChainID       uint64
	Symbol        string // USDT | USDC (what the user sees next to the amount)
	Token         string // token contract, EIP-55
	Decimals      int
	Confirmations uint64 // blocks before a deposit counts
	RPCs          []string
	Explorer      string
	envName       string
}

var networks = []Network{
	{Asset: AssetUSDTEthereum, Symbol: "USDT", Chain: "ethereum", ChainID: 1, Token: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Decimals: 6,
		Confirmations: 12,
		// Public nodes. Some home-network providers silently block a few of these hostnames, so the list is long and mixed.
		RPCs:     []string{"https://ethereum.publicnode.com", "https://eth.drpc.org", "https://eth-mainnet.public.blastapi.io", "https://eth.rpc.blxrbdn.com", "https://ethereum-rpc.publicnode.com", "https://1rpc.io/eth"},
		Explorer: "https://etherscan.io/tx/", envName: "NIMIQ_CLAIM_EVM_RPC_ETHEREUM"},
	// USDC on Polygon: Circle's native USDC (6 decimals, standard ERC-20 that reverts on failure).
	{Asset: AssetUSDCPolygon, Symbol: "USDC", Chain: "polygon", ChainID: 137, Token: "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359", Decimals: 6,
		Confirmations: 64,
		RPCs:          []string{"https://polygon-bor.publicnode.com", "https://matic.drpc.org", "https://polygon.api.onfinality.io/public", "https://polygon-bor-rpc.publicnode.com", "https://1rpc.io/matic", "https://polygon-rpc.com"},
		Explorer:      "https://polygonscan.com/tx/", envName: "NIMIQ_CLAIM_EVM_RPC_POLYGON"},
	{Asset: AssetUSDTPolygon, Symbol: "USDT", Chain: "polygon", ChainID: 137, Token: "0xc2132D05D31c914a87C6611C10748AEb04B58e8F", Decimals: 6,
		Confirmations: 64,
		RPCs:          []string{"https://polygon-bor.publicnode.com", "https://matic.drpc.org", "https://polygon.api.onfinality.io/public", "https://polygon-bor-rpc.publicnode.com", "https://1rpc.io/matic", "https://polygon-rpc.com"},
		Explorer:      "https://polygonscan.com/tx/", envName: "NIMIQ_CLAIM_EVM_RPC_POLYGON"},
}

// Test networks exist only when EnableTestnets is called (a separate test instance: NIMIQ_CLAIM_TESTNET=1). They use free test
// coins, so the whole flow can be exercised on a real chain with real nodes and no money. The token is Circle's official test
// USDC: a standard 6-decimals ERC-20, like the real USDT in every way the cashlink depends on.
const (
	AssetUSDCAmoyTest    = "usdc_amoy_test"
	AssetUSDCSepoliaTest = "usdc_sepolia_test"
)

var testnets = []Network{
	{Asset: AssetUSDCAmoyTest, Symbol: "USDC", Chain: "polygon-amoy-test", ChainID: 80002, Token: "0x41E94Eb019C0762f9Bfcf9Fb1E58725BfB0e7582", Decimals: 6,
		Confirmations: 12,
		RPCs:          []string{"https://polygon-amoy-bor-rpc.publicnode.com", "https://polygon-amoy.drpc.org", "https://rpc-amoy.polygon.technology"},
		Explorer:      "https://amoy.polygonscan.com/tx/", envName: "NIMIQ_CLAIM_EVM_RPC_AMOY"},
	{Asset: AssetUSDCSepoliaTest, Symbol: "USDC", Chain: "ethereum-sepolia-test", ChainID: 11155111, Token: "0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238", Decimals: 6,
		Confirmations: 6,
		RPCs:          []string{"https://ethereum-sepolia-rpc.publicnode.com", "https://sepolia.drpc.org"},
		Explorer:      "https://sepolia.etherscan.io/tx/", envName: "NIMIQ_CLAIM_EVM_RPC_SEPOLIA"},
}

// EnableTestnets adds the test networks. Never called in production.
func EnableTestnets() {
	for _, t := range testnets {
		known := false
		for _, n := range networks {
			known = known || n.Asset == t.Asset
		}
		if !known {
			networks = append(networks, t)
		}
	}
}

var ErrUnknownAsset = errors.New("unknown_asset")

// Lookup returns the network of an asset ID, with RPC endpoints overridden by the environment
// (comma separated, https only) when configured.
func Lookup(asset string) (Network, error) {
	for _, n := range networks {
		if n.Asset == asset {
			if raw := strings.TrimSpace(os.Getenv(n.envName)); raw != "" {
				var list []string
				for _, u := range strings.Split(raw, ",") {
					if u = strings.TrimSpace(u); strings.HasPrefix(u, "https://") {
						list = append(list, u)
					}
				}
				if len(list) > 0 {
					n.RPCs = list
				}
			}
			return n, nil
		}
	}
	return Network{}, ErrUnknownAsset
}

func Assets() []string {
	out := make([]string, 0, len(networks))
	for _, n := range networks {
		out = append(out, n.Asset)
	}
	return out
}

// Symbol is the token symbol shown next to amounts of this asset (USDT when unknown).
func Symbol(asset string) string {
	for _, n := range networks {
		if n.Asset == asset && n.Symbol != "" {
			return n.Symbol
		}
	}
	return "USDT"
}

const (
	MinUnits = 10_000        // 0.01 USDT
	MaxUnits = 1_000_000_000 // 1,000 USDT, same ceiling as a protected NIM payment
)

var ErrInvalidAmount = errors.New("invalid_amount")

// ParseAmount turns "12.5" into 12_500_000 token units (6 decimals) and its canonical text.
func ParseAmount(input string) (units uint64, canonical string, err error) {
	input = strings.TrimSpace(input)
	whole, frac, _ := strings.Cut(input, ".")
	if whole == "" || len(whole) > 7 || len(frac) > 6 || !digits(whole) || (frac != "" && !digits(frac)) || (len(whole) > 1 && whole[0] == '0') || (strings.Contains(input, ".") && frac == "") {
		return 0, "", ErrInvalidAmount
	}
	w, _ := new(big.Int).SetString(whole, 10)
	f := big.NewInt(0)
	if frac != "" {
		f, _ = new(big.Int).SetString(frac+strings.Repeat("0", 6-len(frac)), 10)
	}
	total := new(big.Int).Add(new(big.Int).Mul(w, big.NewInt(1_000_000)), f)
	if !total.IsUint64() || total.Uint64() < MinUnits || total.Uint64() > MaxUnits {
		return 0, "", ErrInvalidAmount
	}
	return total.Uint64(), FormatUnits(total, 6), nil
}

func digits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

// FormatUnits prints units/10^decimals without trailing zeros ("12.5", "3", "0.000123").
func FormatUnits(units *big.Int, decimals int) string {
	s := units.String()
	if len(s) <= decimals {
		s = strings.Repeat("0", decimals-len(s)+1) + s
	}
	whole, frac := s[:len(s)-decimals], strings.TrimRight(s[len(s)-decimals:], "0")
	if frac == "" {
		return whole
	}
	return whole + "." + frac
}
