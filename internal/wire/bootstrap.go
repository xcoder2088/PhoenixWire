package wire

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"

	"github.com/xcoder2088/NimiqSecureClaim/service"
)

// FromEnv starts the embedded engine in the same process. No console-code mode.
// Master keys are generated locally, never printed, and never replaced over state.
func FromEnv() (*API, func(), error) {
	if os.Getenv("PHOENIXWIRE_ENABLE_PAYMENTS") != "1" {
		return nil, func() {}, nil
	}
	delivery := MailerSend{APIKey: os.Getenv("MAILERSEND_API_KEY"), From: os.Getenv("MAILERSEND_FROM")}
	if delivery.Validate() != nil {
		return nil, nil, errors.New("set MAILERSEND_API_KEY and MAILERSEND_FROM")
	}
	origin := os.Getenv("PHOENIXWIRE_PUBLIC_URL")
	if origin == "" {
		return nil, nil, errors.New("set PHOENIXWIRE_PUBLIC_URL to the exact Wire origin")
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, nil, err
	}
	stateHome, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	configDir = filepath.Join(configDir, "phoenixwire")
	stateDir := filepath.Join(stateHome, ".local", "share", "phoenixwire", "state")
	if custom := os.Getenv("PHOENIXWIRE_STATE_DIR"); custom != "" {
		stateDir = custom
	}
	if custom := os.Getenv("PHOENIXWIRE_CONFIG_DIR"); custom != "" {
		configDir = custom
	}
	// Configuration holds key material and is separate from the state backup.
	if err = os.MkdirAll(configDir, 0700); err != nil {
		return nil, nil, err
	}
	info, err := os.Lstat(configDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, nil, errors.New("private_config_directory_required")
	}
	keyPath := filepath.Join(configDir, "master.key")
	var master []byte
	if info, err = os.Lstat(keyPath); errors.Is(err, os.ErrNotExist) {
		// Existing state of any kind blocks key regeneration.
		if files, e := os.ReadDir(stateDir); e != nil && !errors.Is(e, os.ErrNotExist) || len(files) > 0 {
			return nil, nil, errors.New("existing_state_requires_original_master_key")
		}
		master = make([]byte, 32)
		if _, err = rand.Read(master); err != nil {
			return nil, nil, err
		}
		f, e := os.OpenFile(keyPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			clear(master)
			return nil, nil, e
		}
		_, e = f.Write(master)
		if e == nil {
			e = f.Sync()
		}
		closeErr := f.Close()
		if e != nil || closeErr != nil {
			clear(master)
			return nil, nil, errors.New("master_key_write_failed")
		}
	} else {
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return nil, nil, errors.New("private_master_key_required")
		}
		master, err = os.ReadFile(keyPath)
		if err != nil || len(master) != 32 {
			clear(master)
			return nil, nil, errors.New("invalid_master_key")
		}
	}
	defer clear(master)
	rpc := os.Getenv("NIMIQ_CLAIM_MAINNET_RPC_URL")
	if rpc == "" {
		return nil, nil, errors.New("set NIMIQ_CLAIM_MAINNET_RPC_URL")
	}
	engine, err := service.New(service.Config{Directory: filepath.Join(stateDir, "engine"), RPCURL: rpc, Master: digest(master, "Wire engine v1"), Sender: service.MailerSend{APIKey: delivery.APIKey, From: delivery.From, BrandName: "Phoenix Wire"}})
	if err != nil {
		return nil, nil, err
	}
	api, err := New(Config{Directory: filepath.Join(stateDir, "gateway"), Origin: origin, Master: digest(master, "Wire gateway v1"), Engine: engine, Mail: delivery})
	if err != nil {
		engine.Close()
		return nil, nil, err
	}
	return api, func() { api.Close(); engine.Close() }, nil
}
