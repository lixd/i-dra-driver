package device

import (
	"github.com/lixd/i-dra-driver/pkg/common"
	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/dynamic-resource-allocation/resourceslice"
)

func ptrTo(s string) *string {
	return &s
}

func BuildDriverResources(nodeName string, devices []DeviceInfo) resourceslice.DriverResources {
	poolName := nodeName
	driverDevices := make([]resourceapi.Device, 0, len(devices))

	for _, dev := range devices {
		driverDevices = append(driverDevices, resourceapi.Device{
			Name: dev.Name,
			Attributes: map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
				resourceapi.QualifiedName(common.DriverName + "/type"): {
					StringValue: ptrTo(dev.Type),
				},
			},
			Capacity: map[resourceapi.QualifiedName]resourceapi.DeviceCapacity{
				resourceapi.QualifiedName(common.DriverName + "/size"): {
					Value: *resource.NewQuantity(dev.Size, resource.DecimalSI),
				},
			},
		})
	}

	return resourceslice.DriverResources{
		Pools: map[string]resourceslice.Pool{
			poolName: {
				Slices: []resourceslice.Slice{
					{
						Devices: driverDevices,
					},
				},
			},
		},
	}
}
