package iscsi

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fightdou/os-brick-rbd/utils"
	"github.com/wonderivan/logger"
)

// ConnISCSI contains iscsi volume info
type ConnISCSI struct {
	targetDiscovered bool
	targetPortal     string
	targetPortals    []string
	targetIqn        string
	targetIqns       []string
	targetLun        int
	volumeID         bool
	authMethod       string
	authUsername     string
	authPassword     string
	QosSpecs         string
	AccessMode       string
	Encrypted        bool
}

var RetryCount int = 10

type PathIscsi struct {
	Portal string
	Iqn    string
	Lun    int
}

// Hctl is IDs of SCSI
type Hctl struct {
	HostID    int
	ChannelID int
	TargetID  int
	HostLUNID int
}

type SessionIscsi struct {
	Transport            string
	SessionID            int
	TargetPortal         string
	TargetPortalGroupTag int
	IQN                  string
	NodeType             string
}

// NewISCSIConnector Return ConnRbd Pointer to the object
func NewISCSIConnector(connInfo map[string]interface{}) *ConnISCSI {
	data := connInfo["data"].(map[string]interface{})
	conn := &ConnISCSI{}
	conn.targetDiscovered = utils.ToBool(data["target_discovered"])
	conn.targetPortal = utils.ToString(data["target_portal"])
	conn.targetPortals = utils.ToStringSlice(data["target_portal"])
	conn.targetIqn = utils.ToString(data["target_iqn"])
	conn.targetIqns = utils.ToStringSlice(data["target_iqns"])
	conn.targetLun = utils.ToInt(data["target_lun"])
	conn.volumeID = utils.ToBool(data["volume_id"])
	conn.authMethod = utils.ToString(data["auth_method"])
	conn.authUsername = utils.ToString(data["auth_username"])
	conn.authPassword = utils.ToString(data["auth_password"])
	conn.QosSpecs = utils.ToString(data["qos_specs"])
	conn.AccessMode = utils.ToString(data["access_mode"])
	conn.Encrypted = utils.ToBool(data["encrypted"])
	return conn
}

func (c *ConnISCSI) ConnectVolume() (map[string]string, error) {
	var res map[string]string
	if len(c.targetIqns) != 1 {
		c.connectMultiPathVolume()
	} else {
		device, err := c.connectSinglePathVolume()
		if err != nil {
			return nil, err
		}
		res["path"] = device
	}
	return res, nil
}

func (c *ConnISCSI) DisConnectVolume() {

}

func (c *ConnISCSI) ExtendVolume() {

}

func (c *ConnISCSI) GetDevicePath() {

}

func (c *ConnISCSI) connectMultiPathVolume() {

}

func (c *ConnISCSI) connectSinglePathVolume() (string, error) {
	portal, iqn, lun := c.getPortalIqns()
	paths := getIscsiPath(portal, iqn, lun)
	if len(paths) > 1 {
		return "", errors.New("get iscsi path failed")
	}
	p := paths[0]
	device, err := conVolume(p.Portal, p.Iqn, p.Lun)
	if err != nil {
		return "", err
	}
	return filepath.Join("/dev/", device), nil
}

func (c *ConnISCSI) getPortalIqns() ([]string, []string, []int) {
	var portals []string
	var iqns []string
	var luns []int
	args := []string{"-m", "discovery", "-t", "sendtargets", "-p", c.targetPortal}
	out, err := utils.Execute("iscsiadm", args...)
	if err != nil {
		return nil, nil, nil
	}
	entries := strings.Split(out, "\n")
	for _, entry := range entries {
		data := strings.Split(entry, " ")
		if !strings.Contains(data[1], c.targetPortal) {
			continue
		}
		portal := strings.Split(data[0], ",")[0]
		portals = append(portals, portal)
		iqns = append(iqns, data[1])
	}
	for i := 0; i < len(iqns); i++ {
		luns = append(luns, c.targetLun)
	}
	return portals, iqns, luns
}

func getIscsiPath(portals []string, iqns []string, luns []int) []PathIscsi {
	var iscsi []PathIscsi
	for i, _ := range portals {
		p := PathIscsi{
			Portal: portals[i],
			Iqn:    iqns[i],
			Lun:    luns[i],
		}
		iscsi = append(iscsi, p)
	}
	return iscsi
}

func conVolume(portal string, iqn string, lun int) (string, error) {
	sessionId, err := connectToIscsiPortal(portal, iqn)
	if err != nil {
		return "", err
	}
	hctl, err := getHctl(sessionId, lun)
	if err != nil {
		return "", err
	}
	if err := scanISCSI(hctl); err != nil {
		return "", fmt.Errorf("failed to rescan target: %w", err)
	}
	device, err := GetDeviceName(sessionId, hctl)
	if err != nil {
		return "", fmt.Errorf("failed to get device name: %w", err)
	}
	return device, nil

}
func GetDeviceName(id int, hctl *Hctl) (string, error) {
	p := fmt.Sprintf(
		"/sys/class/iscsi_host/host%d/device/session%d/target%d:%d:%d/%d:%d:%d:%d/block/*",
		hctl.HostID,
		id,
		hctl.HostID, hctl.ChannelID, hctl.TargetID,
		hctl.HostID, hctl.ChannelID, hctl.TargetID, hctl.HostLUNID)

	paths, err := filepath.Glob(p)
	if err != nil {
		return "", fmt.Errorf("failed to parse iSCSI block device filepath: %w", err)
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("device filepath is not found")
	}

	_, deviceName := filepath.Split(paths[0])

	return filepath.Join("/dev/", deviceName), nil
}

func connectToIscsiPortal(portal string, iqn string) (int, error) {
	if err := loginPortal(portal, iqn); err != nil {
		logger.Error("Iscsi login portal failed", err)
		return 0, err
	}
	for i := 0; i < RetryCount; i++ {
		sessions, err := getSessions()
		if err != nil {
			logger.Error("Get iscsi session failed", err)
			return 0, err
		}
		for _, session := range sessions {
			if session.TargetPortal == portal && session.IQN == iqn {
				return session.SessionID, nil
			}
		}
	}
	return -1, errors.New("session id not found")
}

func loginPortal(portal string, iqn string) error {
	_, err := utils.ExecIscsiadm(portal, iqn, []string{"--login"})
	if err != nil {
		logger.Error("Exec iscsiadm login command failed", err)
		return err
	}
	_, err = utils.UpdateIscsiadm(portal, iqn, "node.startup", "automatic", nil)
	if err != nil {
		logger.Error("Exec iscsiadm update command failed", err)
		return err
	}
	logger.Info("iscsiadm portal %s login success", portal)
	return nil
}

func getSessions() ([]SessionIscsi, error) {
	args := []string{"-m", "session"}
	out, err := utils.Execute("iscsiadm", args...)
	if err != nil {
		logger.Error("Exec iscsiadm -m session command failed", err)
		return nil, err
	}
	session, err := parseSession(out)
	if err != nil {
		logger.Error("Parse session info failed", err)
		return nil, err
	}
	return session, nil
}

func parseSession(out string) ([]SessionIscsi, error) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	re := strings.NewReplacer("[", "", "]", "")
	var session []SessionIscsi
	for _, line := range lines {
		l := strings.Fields(line)
		if len(l) < 4 {
			continue
		}
		protocol := strings.Split(l[0], ":")[0]
		id := re.Replace(l[1])
		id64, _ := strconv.ParseInt(id, 10, 32)
		portal := strings.Split(l[2], ",")[0]
		portalTag, err := strconv.Atoi(strings.Split(l[2], ",")[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse portal port group tag: %w", err)
		}
		s := SessionIscsi{
			Transport:            protocol,
			SessionID:            int(id64),
			TargetPortal:         portal,
			TargetPortalGroupTag: portalTag,
			IQN:                  l[3],
			NodeType:             strings.Split(l[3], ":")[1],
		}
		session = append(session, s)

	}
	return session, nil
}

func getHctl(id int, lun int) (*Hctl, error) {
	globStr := fmt.Sprintf("/sys/class/iscsi_host/host*/device/session%d/target*", id)
	paths, err := filepath.Glob(globStr)
	if err != nil {
		logger.Error("Failed to get session path", err)
		return nil, err
	}
	if len(paths) != 1 {
		logger.Error("target fail is not found", err)
		return nil, err
	}
	_, fileName := filepath.Split(paths[0])
	ids := strings.Split(fileName, ":")
	if len(ids) != 3 {
		return nil, fmt.Errorf("failed to parse iSCSI session filename")
	}
	channelID, err := strconv.Atoi(ids[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse channel ID: %w", err)
	}
	targetID, err := strconv.Atoi(ids[2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse target ID: %w", err)
	}

	names := strings.Split(paths[0], "/")
	hostIDstr := strings.TrimPrefix(searchHost(names), "host")
	hostID, err := strconv.Atoi(hostIDstr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse host ID: %w", err)
	}

	hctl := &Hctl{
		HostID:    hostID,
		ChannelID: channelID,
		TargetID:  targetID,
		HostLUNID: lun,
	}

	return hctl, nil
}

// searchHost search param
// return "host"+id
func searchHost(names []string) string {
	for _, v := range names {
		if strings.HasPrefix(v, "host") {
			return v
		}
	}

	return ""
}

func scanISCSI(hctl *Hctl) error {
	path := fmt.Sprintf("/sys/class/scsi_host/host%d/scan", hctl.HostID)
	content := fmt.Sprintf("%d %d %d",
		hctl.ChannelID,
		hctl.TargetID,
		hctl.HostLUNID)

	return utils.EchoScsiCommand(path, content)
}
