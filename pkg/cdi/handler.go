package cdi

import (
	"fmt"
	"os"
	"strings"

	"k8s.io/klog/v2"
	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"
	cdispec "tags.cncf.io/container-device-interface/specs-go"
)

type Handler struct {
	cache      *cdiapi.Cache
	driverName string
	class      string
	vendor     string
	devicePath string
}

func NewHandler(driverName, vendor, class, devicePath string) (*Handler, error) {
	cdiDir := "/var/run/cdi"
	if err := os.MkdirAll(cdiDir, 0755); err != nil {
		return nil, fmt.Errorf("create CDI directory %s failed: %w", cdiDir, err)
	}

	cache, err := cdiapi.NewCache(
		cdiapi.WithSpecDirs(cdiDir),
	)
	if err != nil {
		return nil, fmt.Errorf("create CDI cache failed: %w", err)
	}

	return &Handler{
		cache:      cache,
		driverName: driverName,
		class:      class,
		vendor:     vendor,
		devicePath: devicePath,
	}, nil
}

func (h *Handler) kind() string {
	return fmt.Sprintf("%s/%s", h.vendor, h.class)
}

func (h *Handler) CreateClaimSpec(claimUID string, deviceNames []string) ([]string, error) {
	spec := &cdispec.Spec{
		Kind:    h.kind(),
		Devices: []cdispec.Device{},
	}

	var cdiDeviceIDs []string

	for _, devName := range deviceNames {
		cdiDeviceName := fmt.Sprintf("%s-%s", claimUID, devName)
		cdiDeviceID := fmt.Sprintf("%s/%s=%s", h.vendor, h.class, cdiDeviceName)
		cdiDeviceIDs = append(cdiDeviceIDs, cdiDeviceID)

		device := cdispec.Device{
			Name: cdiDeviceName,
			ContainerEdits: cdispec.ContainerEdits{
				Env: []string{
					fmt.Sprintf("GOPHER=%s", strings.Join(deviceNames, ",")),
				},
			},
		}
		spec.Devices = append(spec.Devices, device)
	}

	minVersion, err := cdiapi.MinimumRequiredVersion(spec)
	if err != nil {
		return nil, fmt.Errorf("get CDI minimum version failed: %w", err)
	}
	spec.Version = minVersion

	specName := cdiapi.GenerateTransientSpecName(h.vendor, h.class, claimUID)
	if err := h.cache.WriteSpec(spec, specName); err != nil {
		return nil, fmt.Errorf("write CDI spec failed: %w", err)
	}

	klog.Infof("created CDI spec for claim %s, devices: %v", claimUID, deviceNames)
	return cdiDeviceIDs, nil
}

func (h *Handler) DeleteClaimSpec(claimUID string) error {
	specName := cdiapi.GenerateTransientSpecName(h.vendor, h.class, claimUID)
	err := h.cache.RemoveSpec(specName)
	if err != nil {
		if os.IsNotExist(err) {
			klog.Infof("CDI spec for claim %s already removed", claimUID)
			return nil
		}
		return fmt.Errorf("remove CDI spec for claim %s failed: %w", claimUID, err)
	}

	klog.Infof("deleted CDI spec for claim %s", claimUID)
	return nil
}
