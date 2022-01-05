package utils

import (
	"fmt"
	"os/exec"
	"strconv"
)

// Execute a shell command
func Execute(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	stdoutStderr, err := cmd.CombinedOutput()
	return string(stdoutStderr), err
}

func ToBool(i interface{}) bool {
	switch res := i.(type) {
	case bool:
		return res
	case int, int32, int64, uint, uint32, uint64:
		return res != 0
	case string:
		result, _ := strconv.ParseBool(res)
		return result
	default:
		panic("Can not transition interface to bool")
	}
}

func ToString(i interface{}) string {
	res := fmt.Sprint(i)
	return res
}

func ToStringSlice(i interface{}) []string {
	resSlice := i.([]interface{})
	result := make([]string, len(resSlice))
	for i, v := range resSlice {
		result[i] = v.(string)
	}
	return result
}
