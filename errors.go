package openvpn

import "errors"

var (
	ErrAlreadyRunning   = errors.New("VPN client is already running")
	ErrTimeout          = errors.New("connection timed out or context cancelled")
	ErrBinaryNotFound   = errors.New("openvpn binary not found in PATH")
	ErrConnectionFailed = errors.New("openvpn process exited with error")
)
