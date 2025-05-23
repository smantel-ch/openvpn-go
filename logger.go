package openvpn

type Logger interface {
	Debugf(format string, args ...any)
}

var logger Logger = nil

func SetLogger(l Logger) {
	logger = l
}
