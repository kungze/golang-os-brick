package local

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kungze/golang-os-brick/utils"
	"github.com/wonderivan/logger"
)

type ConnLocal struct {
	volumeID   string
}

func NewLocalConnector(connInfo map[string]interface{}) *ConnLocal {
	conn := &ConnLocal{}
	conn.volumeID = utils.ToString(connInfo["volume_id"])
	return conn
}

func (c *ConnLocal) ConnectVolume() (map[string]string, error) {
	res := map[string]string{}
	globStr := fmt.Sprintf("/dev/*/*%s", c.volumeID)
	paths, err := filepath.Glob(globStr)
	if err != nil {
		return nil, err
	}
	if len(paths) != 1 {
		logger.Error("lvm volume path not found", err)
		return nil, err
	}
	logger.Info("Get lvm path success", paths[0])
	res["path"] = paths[0]
	return res, nil
}

func (c *ConnLocal) DisConnectVolume() {
	logger.Info("local volume disconnect volume success")
}

func (c *ConnLocal) ExtendVolume() (int64, error) {
	globStr := fmt.Sprintf("/dev/*/*%s", c.volumeID)
	paths, err := filepath.Glob(globStr)
	if err != nil {
		return 0, err
	}
	if len(paths) != 1 {
		logger.Error("lvm volume path not found", err)
		return 0, err
	}
	sizeCmd := fmt.Sprintf("lvdisplay --units B %s 2>&1 | grep 'LV Size' | awk '{print $3}'", globStr)
	out, err := utils.Execute(sizeCmd)
	if err != nil {
		logger.Error("Exec lvdisplay command failed", err)
		return 0, err
	}
	sizeStr := strings.Split(out, ".")[0]
	sizeInt, err := strconv.ParseInt(strings.TrimSpace(sizeStr), 10, 64)
	if err != nil {
		logger.Error("Parse lvm size failed", err)
		return 0, err
	}
	logger.Info("Get lvm size success", sizeInt)
	return sizeInt, nil
}

func (c *ConnLocal) GetDevicePath() string {
	globStr := fmt.Sprintf("/dev/*/*%s", c.volumeID)
	paths, err := filepath.Glob(globStr)
	if err != nil {
		return ""
	}
	if len(paths) != 1 {
		logger.Error("lvm volume path not found", err)
		return ""
	}
	logger.Info("Get lvm path success", paths[0])
	return paths[0]
}
