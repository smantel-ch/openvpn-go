package openvpn

import (
	"errors"
	"strings"
)

var (
	ErrAlreadyRunning     = errors.New("VPN client is already running")
	ErrTimeout            = errors.New("connection timed out or context cancelled")
	ErrBinaryNotFound     = errors.New("openvpn binary not found in PATH")
	ErrConnectionFailed   = errors.New("openvpn process exited with error")
	ErrZombieProcess      = errors.New("openvpn process already running and could not be killed")
	ErrDisconnectFailed   = errors.New("openvpn process could not confirm disconnect")
	ErrInvalidCredentials = errors.New("authentication failed")
)

var knownErrorMatchers = map[string]error{
	"AUTH_FAILED":                  ErrInvalidCredentials,
	"auth-failure":                 ErrInvalidCredentials,
	"Connection refused":           ErrConnectionFailed,
	"Permission denied":            ErrConnectionFailed,
	"TLS Error":                    ErrConnectionFailed,
	"error parsing ca certificate": ErrConnectionFailed,
	"Cannot load certificate file": ErrConnectionFailed,
}

func detectLogError(line string) error {
	for key, err := range knownErrorMatchers {
		if strings.Contains(line, key) {
			logger.Debugf("Matched log line: %s", line)
			return err
		}
	}
	return nil
}
