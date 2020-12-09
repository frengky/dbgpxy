package dbgpxy

import "fmt"

// IDE represent editor that listening for XDebug connection
type IDE interface {
	GetKey() string
	GetAddress() string
}

// NewIDE create new IDE associated with key, ip and port
func NewIDE(key string, ip string, port string) IDE {
	return &defaultIDE{
		Key:  key,
		IP:   ip,
		Port: port,
	}
}

type defaultIDE struct {
	Key  string
	IP   string
	Port string
}

func (i *defaultIDE) GetKey() string {
	return i.Key
}

func (i *defaultIDE) GetAddress() string {
	return fmt.Sprintf("%s:%s", i.IP, i.Port)
}
