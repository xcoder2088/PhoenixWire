//go:build windows

package service

import "errors"

func lockPrivateState(string) (func(), error) {
	return nil, errors.New("SecureClaim durable server currently requires Linux")
}
