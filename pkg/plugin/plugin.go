package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lixd/i-dra-driver/pkg/cdi"
	"github.com/lixd/i-dra-driver/pkg/common"
	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/dynamic-resource-allocation/kubeletplugin"
)

type GopherDRAPlugin struct {
	cdiHandler *cdi.Handler
	mu         sync.Mutex
}

func NewGopherDRAPlugin(cdiHandler *cdi.Handler) *GopherDRAPlugin {
	return &GopherDRAPlugin{
		cdiHandler: cdiHandler,
	}
}

func (p *GopherDRAPlugin) PrepareResourceClaims(ctx context.Context, claims []*resourceapi.ResourceClaim) (map[types.UID]kubeletplugin.PrepareResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make(map[types.UID]kubeletplugin.PrepareResult)

	for _, claim := range claims {
		claimUID := claim.UID
		klog.Infof("PrepareResourceClaims: claim=%s/%s uid=%s", claim.Namespace, claim.Name, claimUID)

		if claim.Status.Allocation == nil {
			klog.Warningf("claim %s/%s is not allocated, skipping", claim.Namespace, claim.Name)
			result[claimUID] = kubeletplugin.PrepareResult{}
			continue
		}

		deviceNames := p.extractAllocatedDevices(claim)
		if len(deviceNames) == 0 {
			klog.Warningf("no gopher devices found in claim %s/%s", claim.Namespace, claim.Name)
			result[claimUID] = kubeletplugin.PrepareResult{}
			continue
		}

		cdiDeviceIDs, err := p.cdiHandler.CreateClaimSpec(string(claimUID), deviceNames)
		if err != nil {
			klog.Errorf("create CDI spec for claim %s/%s failed: %v", claim.Namespace, claim.Name, err)
			result[claimUID] = kubeletplugin.PrepareResult{
				Err: fmt.Errorf("create CDI spec failed: %w", err),
			}
			continue
		}

		devices := make([]kubeletplugin.Device, 0)
		for _, req := range claim.Status.Allocation.Devices.Results {
			if req.Driver != common.DriverName {
				continue
			}
			devices = append(devices, kubeletplugin.Device{
				Requests:    []string{req.Request},
				PoolName:    req.Pool,
				DeviceName:  req.Device,
				CDIDeviceIDs: cdiDeviceIDs,
			})
		}

		result[claimUID] = kubeletplugin.PrepareResult{
			Devices: devices,
		}
		klog.Infof("prepared claim %s/%s with CDI devices: %v", claim.Namespace, claim.Name, cdiDeviceIDs)
	}

	return result, nil
}

func (p *GopherDRAPlugin) UnprepareResourceClaims(ctx context.Context, claims []kubeletplugin.NamespacedObject) (map[types.UID]error, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make(map[types.UID]error)

	for _, claim := range claims {
		claimUID := claim.UID
		klog.Infof("UnprepareResourceClaims: claim=%s/%s uid=%s", claim.Namespace, claim.Name, claimUID)

		if err := p.cdiHandler.DeleteClaimSpec(string(claimUID)); err != nil {
			klog.Errorf("delete CDI spec for claim %s/%s failed: %v", claim.Namespace, claim.Name, err)
			result[claimUID] = err
			continue
		}

		result[claimUID] = nil
		klog.Infof("unprepared claim %s/%s", claim.Namespace, claim.Name)
	}

	return result, nil
}

func (p *GopherDRAPlugin) HandleError(ctx context.Context, err error, msg string) {
	if errors.Is(err, kubeletplugin.ErrRecoverable) {
		klog.Warningf("recoverable error: %s: %v", msg, err)
		return
	}
	klog.Fatalf("fatal error: %s: %v", msg, err)
}

func (p *GopherDRAPlugin) extractAllocatedDevices(claim *resourceapi.ResourceClaim) []string {
	if claim.Status.Allocation == nil {
		return nil
	}

	var deviceNames []string
	for _, result := range claim.Status.Allocation.Devices.Results {
		if result.Driver == common.DriverName {
			deviceNames = append(deviceNames, result.Device)
		}
	}
	return deviceNames
}
