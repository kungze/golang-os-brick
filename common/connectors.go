package common

import (
	"github.com/kungze/golang-os-brick/initiator"
	"github.com/kungze/golang-os-brick/initiator/connectors/rbd"
	"strings"
)

// NewConnector Build a Connector object based upon protocol and architecture
func NewConnector(protocol string, connInfo map[string]interface{}) initiator.ConnProperties {
	switch strings.ToUpper(protocol) {
	case "RBD":
		connInfo["do_local_attach"] = true
		connRbd := rbd.NewRBDConnector(connInfo)
		return connRbd
	}
	return nil
}
