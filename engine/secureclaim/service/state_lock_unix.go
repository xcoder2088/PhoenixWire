//go:build !windows

package service

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

func lockPrivateState(directory string) (func(), error) {
	if !filepath.IsAbs(directory) {
		return nil, errors.New("absolute private state path required")
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	st, err := os.Lstat(directory)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 {
		return nil, errors.New("private state directory required")
	}
	path := filepath.Join(directory, ".process.lock")
	if st, err := os.Lstat(path); err == nil && (!st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0) {
		return nil, errors.New("private process lock required")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, err
	}
	return func() { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }, nil
}
