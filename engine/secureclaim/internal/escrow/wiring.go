package escrow

import (
	"errors"
	"fmt"

	"github.com/xcoder2088/NimiqSecureClaim/internal/claim"
	"github.com/xcoder2088/NimiqSecureClaim/internal/nimiq"
	"github.com/xcoder2088/NimiqSecureClaim/internal/vault"
)

// NewMainnetAdmission wires a trusted Nimiq Mainnet RPC verifier, independent
// Cashlink seed->address derivation, and encrypted local vault together.
// This is internal service setup ONLY. It does not create a browser route,
// issue funding transactions, or release funds after OTP.
func NewMainnetAdmission(store *claim.Store, rpcURL, vaultDirectory, masterHex, keyID string) (*Admission, error) {
	if store == nil || rpcURL == "" || vaultDirectory == "" || masterHex == "" || keyID == "" {
		return nil, ErrUnavailable
	}
	verifier, err := nimiq.NewMainnetVerifier(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid mainnet RPC configuration", ErrUnavailable)
	}
	master, err := vault.DecodeMasterHex(masterHex)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid vault master configuration", ErrUnavailable)
	}
	v, err := vault.New(master, keyID)
	clear(master)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid vault key label", ErrUnavailable)
	}
	private, err := vault.NewFileStore(vaultDirectory, v)
	if err != nil {
		v.Close()
		return nil, fmt.Errorf("%w: private vault unavailable", ErrUnavailable)
	}
	admission, err := NewAdmission(store, verifier, nimiq.AddressDeriver{}, private)
	if err != nil {
		v.Close()
		return nil, errors.Join(ErrUnavailable, err)
	}
	return admission, nil
}

// NewMainnetProvisioner wires the secure pre-funding path. A Cashlink seed
// is generated and encrypted before the sender sees the public funding address.
// No route is installed and no payout capability is granted.
func NewMainnetProvisioner(store *claim.Store, rpcURL, vaultDirectory, masterHex, keyID string) (*Provisioner, error) {
	if store == nil || rpcURL == "" || vaultDirectory == "" || masterHex == "" || keyID == "" {
		return nil, ErrUnavailable
	}
	verifier, err := nimiq.NewMainnetVerifier(rpcURL)
	if err != nil {
		return nil, ErrUnavailable
	}
	master, err := vault.DecodeMasterHex(masterHex)
	if err != nil {
		return nil, ErrUnavailable
	}
	v, err := vault.New(master, keyID)
	clear(master)
	if err != nil {
		return nil, ErrUnavailable
	}
	private, err := vault.NewFileStore(vaultDirectory, v)
	if err != nil {
		v.Close()
		return nil, ErrUnavailable
	}
	p, err := NewProvisioner(store, nimiq.AddressDeriver{}, private, verifier)
	if err != nil {
		v.Close()
		return nil, ErrUnavailable
	}
	return p, nil
}
