package openvpn

type VPNStatus string

const (
	StatusInitializing VPNStatus = "initializing"
	StatusConnected    VPNStatus = "connected"
	StatusDisconnected VPNStatus = "disconnected"
	StatusError        VPNStatus = "error"
)
