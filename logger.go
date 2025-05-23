package openvpn

type Logger interface {
	Debugf(format string, args ...interface{})
}

var moduleLogger Logger = nil

func SetLogger(l Logger) {
	moduleLogger = l
}
