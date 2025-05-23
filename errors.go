package openvpn

import (
	"errors"
	"strings"
)

type VPNError struct {
	Code    error
	Message string
}

func (e VPNError) Error() string {
	return e.Message
}

func (e VPNError) Unwrap() error {
	return e.Code
}

var (
	ErrAlreadyRunning   = errors.New("VPN client is already running")
	ErrTimeout          = errors.New("connection timed out or context cancelled")
	ErrBinaryNotFound   = errors.New("openvpn binary not found in PATH")
	ErrConnectionFailed = errors.New("openvpn process exited with error")
	ErrZombieProcess    = errors.New("openvpn process already running and could not be killed")
)

func detectLogError(line string) string {
	if moduleLogger != nil {
		moduleLogger.Debugf("[VPN] Matched log line: %s", line)
	}

	lower := strings.ToLower(line)

	switch {
	case strings.Contains(lower, "auth_failed"):
		return "Authentication failed"
	case strings.Contains(lower, "tls handshake failed"):
		return "TLS handshake failed — possibly a server, firewall, or cert issue"
	case strings.Contains(lower, "cannot resolve host address"):
		return "DNS resolution failed — check VPN server hostname"
	case strings.Contains(lower, "connection timed out"):
		return "Connection timed out"
	case strings.Contains(lower, "no route to host"):
		return "No route to host — network or firewall issue"
	case strings.Contains(lower, "inactivity timeout"):
		return "Connection dropped due to inactivity"
	case strings.Contains(lower, "socket bind failed"):
		return "Port already in use — try a different local port"
	case strings.Contains(lower, "tls error"):
		return "Generic TLS error"
	case strings.Contains(lower, "error") || strings.Contains(lower, "fatal"):
		return line
	default:
		return ""
	}
}
