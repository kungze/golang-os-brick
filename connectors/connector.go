package connectors

import (
	"github.com/kungze/golang-os-brick/connectors/rbd"
	"strings"
)

// ConnProperties is base class interface
type ConnProperties interface {
	ConnectVolume() (map[string]string, error)
	DisConnectVolume()
	ExtendVolume() (int64, error)
        GetDevicePath() string
}

// NewConnector Build a Connector object based upon protocol and architecture
func NewConnector(protocol string, connInfo map[string]interface{}) ConnProperties {
	switch strings.ToUpper(protocol) {
	case "RBD":
		// Only supported local attach volume
		connInfo["do_local_attach"] = true
		return rbd.NewRBDConnector(connInfo)
	}
	return nil
}
