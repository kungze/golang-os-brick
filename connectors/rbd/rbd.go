package rbd

import (
	"encoding/json"
	"fmt"
	"github.com/kungze/golang-os-brick/utils"
	"github.com/wonderivan/logger"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
)

// ConnRbd contains rbd volume info
type ConnRbd struct {
	Name          string
	Hosts         []string
	Ports         []string
	ClusterName   string
	AuthEnabled   bool
	AuthUserName  string
	VolumeID      string
	Discard       bool
	QosSpecs      string
	Keyring       string
	AccessMode    string
	Encrypted     bool
	DoLocalAttach bool
}

// NewRBDConnector Return ConnRbd Pointer to the object
func NewRBDConnector(connInfo map[string]interface{}) *ConnRbd {
	data := connInfo["data"].(map[string]interface{})
	conn := &ConnRbd{}
	conn.Name = utils.ToString(data["name"])
	conn.Hosts = utils.ToStringSlice(data["hosts"])
	conn.Ports = utils.ToStringSlice(data["ports"])
	conn.ClusterName = utils.ToString(data["cluster_name"])
	conn.AuthEnabled = utils.ToBool(data["auth_enabled"])
	conn.AuthUserName = utils.ToString(data["auth_username"])
	conn.VolumeID = utils.ToString(data["volume_id"])
	conn.Discard = utils.ToBool(data["discard"])
	conn.QosSpecs = utils.ToString(data["qos_specs"])
	conn.AccessMode = utils.ToString(data["access_mode"])
	conn.Encrypted = utils.ToBool(data["encrypted"])
	conn.DoLocalAttach = utils.ToBool(connInfo["do_local_attach"])
	return conn
}

// ConnectVolume Connect to a volume
func (c *ConnRbd) ConnectVolume() (map[string]string, error) {
	var err error
	result := map[string]string{}
	// Only supported local attach volume
	if c.DoLocalAttach {
		result, err = c.localAttachVolume()
		if err != nil {
			logger.Error("Do local attach volume failed", err)
			return nil, err
		}
		logger.Info("Rbd Connect Success, Map Path is %s", result)
		return result, nil
	}
	return nil, err
}

// DisConnectVolume Disconnect a volume
func (c *ConnRbd) DisConnectVolume() {
	// Only supported local attach volume
	if c.DoLocalAttach {
		rootDevice := c.findRootDevice()
		if rootDevice != "" {
			cmd := []string{"unmap", rootDevice}
			utils.Execute("rbd", cmd...)
			logger.Info("Exec rbd unmap command success")
		}
	}
}

// ExtendVolume Refresh local volume view and return current size in bytes
// Nothing to do, RBD attached volumes are automatically refreshed, but
// we need to return the new size for compatibility
func (c *ConnRbd) ExtendVolume() (int64, error) {
	var err error
	// Only supported local attach volume
	if c.DoLocalAttach {
		device := c.findRootDevice()
		if device == "" {
			logger.Error("device is not exist", err)
			return 0, err
		}
		deviceName := path.Base(device)
		deviceNumber := deviceName[3:]
		size, err := ioutil.ReadFile("/sys/devices/rbd/" + deviceNumber + "/size")
		if err != nil {
			logger.Error("Read /sys/devices/rbd/?/size failed", err)
			return 0, err
		}
		strSize := string(size)
		vSize := strings.Replace(strSize, "'", "", -1)
		iSize, _ := strconv.ParseInt(vSize, 10, 64)
		logger.Info("extend volume to %s is success", iSize)
		return iSize, nil
	}
	return 0, err
}

// findRootDevice Find the underlying /dev/rbd* device for a mapping
// Use the showmapped command to list all acive mappings and find the
// underlying /dev/rbd* device that corresponds to our pool and volume
func (c *ConnRbd) findRootDevice() string {
	volume := strings.Split(c.Name, "/")
	poolVolume := volume[1]
	cmd := []string{"showmapped", "--format=json"}
	cmd = append(cmd, "--id")
	cmd = append(cmd, c.AuthUserName)
	res, err := utils.Execute("rbd", cmd...)
	logger.Debug("Exec rbd showmapped command success", res)
	if err != nil {
		logger.Error("Exec rbd showmapped failed", err)
		return ""
	}
	var result []map[string]string
	err = json.Unmarshal([]byte(res), &result)
	if err != nil {
		logger.Error("conversion json failed")
		return ""
	}
	for _, mapping := range result {
		if mapping["name"] == poolVolume {
			return mapping["device"]
		}
	}
	return ""
}

// localAttachVolume Exec local attach volume process
func (c *ConnRbd) localAttachVolume() (map[string]string, error) {
	res := map[string]string{}
	out, err := utils.Execute("which", "rbd")
	if err != nil {
		logger.Error("Exec which rbd command failed", err)
		return nil, err
	}
	if out == "" {
		logger.Error("ceph-common package is not installed")
		return nil, err
	}

	volume := strings.Split(c.Name, "/")
	poolName := volume[0]
	poolVolume := volume[1]
	rbdDevPath := getRbdDeviceName(poolName, poolVolume)
	_, err = os.Readlink(rbdDevPath)
	if err != nil {
		cmd := []string{"map", poolVolume, "--pool", poolName, "--id", c.AuthUserName}
		result, err := utils.Execute("rbd", cmd...)
		logger.Info("command succeeded: rbd map path is %s", result)
		if err != nil {
			logger.Error("rbd map command exec failed", err)
			return nil, err
		}
	} else {
		logger.Info("Volume %s is already mapped to local device %s", poolVolume, rbdDevPath)
		return nil, err
	}

	res["path"] = rbdDevPath
	res["type"] = "block"
	return res, nil
}

// getRbdDeviceName Return device name which will be generated by RBD kernel module
func getRbdDeviceName(pool string, volume string) string {
	return fmt.Sprintf("/dev/rbd/%s/%s", pool, volume)
}
