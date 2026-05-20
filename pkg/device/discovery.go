package device

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"k8s.io/klog/v2"
)

type DeviceInfo struct {
	Name string
	Type string
	Size int64
}

func Discover(devicePath string) ([]DeviceInfo, error) {
	var devices []DeviceInfo

	info, err := os.Stat(devicePath)
	if err != nil {
		if os.IsNotExist(err) {
			klog.Warningf("device path %s does not exist, publishing empty device list", devicePath)
			return devices, nil
		}
		return nil, fmt.Errorf("stat device path %s failed: %w", devicePath, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("device path %s is not a directory", devicePath)
	}

	err = filepath.WalkDir(devicePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s failed: %w", path, err)
		}
		if d.IsDir() {
			if path != devicePath {
				return filepath.SkipDir
			}
			return nil
		}

		fi, err := d.Info()
		if err != nil {
			klog.Warningf("get file info for %s failed: %v", path, err)
			return nil
		}

		devices = append(devices, DeviceInfo{
			Name: d.Name(),
			Type: "gopher",
			Size: fi.Size(),
		})
		klog.Infof("discovered device: %s (size=%d)", d.Name(), fi.Size())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk device path %s failed: %w", devicePath, err)
	}

	if len(devices) == 0 {
		klog.Infof("no devices found in %s", devicePath)
	}

	return devices, nil
}
