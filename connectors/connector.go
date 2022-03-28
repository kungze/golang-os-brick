package connectors

import (
	"strings"

	"github.com/kungze/golang-os-brick/connectors/local"
	"github.com/kungze/golang-os-brick/rbd"
)

// ConnProperties is base class interface
type ConnProperties interface {
	ConnectVolume() (map[string]string, error)
	DisConnectVolume() error
	ExtendVolume() (int64, error)
	GetDevicePath() string
}

// NewConnector Build a Connector object based upon protocol and architecture
func NewConnector(protocol string, connInfo map[string]interface{}) ConnProperties {
	switch strings.ToUpper(protocol) {
	case "Local":
		return local.NewLocalConnector(connInfo)
	case "RBD":
		// Only supported local attach volume
		connInfo["do_local_attach"] = true
		return rbd.NewRBDConnector(connInfo)
	}
	return nil
}
