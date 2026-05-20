# i-dra-driver

A custom Kubernetes DRA (Dynamic Resource Allocation) driver that exposes files as "gopher" devices.

This is a demo project for the blog post [从零开发自己的 DRA 驱动](https://www.lixueduan.com/posts/kubernetes/58-dra-p4-my-dra-driver/), designed to compare DRA with the legacy [DevicePlugin](https://www.lixueduan.com/posts/kubernetes/21-device-plugin/) approach.

## How It Works

The driver scans files under `/etc/gophers` on each node and exposes each file as a "gopher" device via DRA:

- **Device Discovery**: scans `/etc/gophers`, each file becomes a device with `type=gopher` and `size=<file-size>` attributes
- **ResourceSlice**: publishes device inventory (with attributes and capacity) to the API server, visible to the scheduler
- **DeviceClass**: CEL selector matching `device.driver == 'gopher.example.com' && device.attributes['gopher.example.com'].type == 'gopher'`
- **CDI Injection**: on `NodePrepareResources`, creates CDI specs that inject `GOPHER=<device-names>` env var into containers

## Prerequisites

- Kubernetes 1.32+ with DRA feature gate enabled (v1.34+ has it enabled by default)
- Container runtime with CDI support (containerd v1.7+ enables CDI by default)

## Quick Start

### 1. Prepare device files on nodes

```bash
mkdir -p /etc/gophers
echo "hello from gopher-a" > /etc/gophers/gopher-a
echo "hello from gopher-b" > /etc/gophers/gopher-b
```

### 2. Deploy the driver

```bash
# RBAC
kubectl apply -f deploy/rbac.yaml

# DaemonSet
kubectl apply -f deploy/daemonset.yaml

# DeviceClass
kubectl apply -f deploy/deviceclass.yaml
```

> **Note**: The plugin socket directory `/var/lib/kubelet/plugins/gopher.example.com/` must exist on the host before the DaemonSet starts. You can create it with: `mkdir -p /var/lib/kubelet/plugins/gopher.example.com/`

### 3. Verify ResourceSlice

```bash
kubectl get resourceslice | grep gopher
```

You should see a ResourceSlice for each node with gopher devices:

```
NAME                                        NODE         DRIVER                  POOL         AGE
00000-gopher.example.com-ecs-a10-sh-nj7lm   ecs-a10-sh   gopher.example.com      ecs-a10-sh   18s
```

Check device details:

```bash
kubectl get resourceslice <name> -oyaml
```

### 4. Run a test Pod

```bash
kubectl apply -f deploy/test-pod.yaml
```

Verify:

```bash
# Pod should be Running
kubectl get pod gopher-test-pod

# Check device injection
kubectl logs gopher-test-pod
# Expected output:
# GOPHER env: gopher-a
# Gopher device allocated successfully!
# Device file: /etc/gophers/gopher-a
# hello from gopher-a

# Verify env var inside container
kubectl exec gopher-test-pod -- env | grep GOPHER
# GOPHER=gopher-a
```

### 5. Check ResourceClaim allocation

```bash
kubectl get resourceclaim
kubectl get resourceclaim <claim-name> -oyaml
```

The status should show `allocated,reserved` with the specific device assigned by the scheduler.

## Build

```bash
# Build binary
make build

# Build Docker image
make build-image
```

## Project Structure

```
i-dra-driver/
  cmd/main.go                 # Entry point
  pkg/
    common/constant.go        # Constants (driver name, device path, CDI vendor)
    device/
      discovery.go            # Device discovery: scan /etc/gophers
      resourceslice.go        # Build DriverResources for ResourceSlice
    cdi/handler.go            # CDI spec management: create/delete
    plugin/plugin.go          # DRAPlugin interface implementation
  deploy/
    rbac.yaml                 # ServiceAccount + ClusterRole + ClusterRoleBinding
    daemonset.yaml            # DaemonSet deployment
    deviceclass.yaml          # DeviceClass with CEL selector
    test-pod.yaml             # ResourceClaimTemplate + test Pod
```

## Related

- [i-device-plugin](https://github.com/lixd/i-device-plugin) — The DevicePlugin version of this project
- [DRA P1: Quickstart](https://www.lixueduan.com/posts/kubernetes/54-dra-p1-quickstart/)
- [DRA P2: Slice, Class, Claim](https://www.lixueduan.com/posts/kubernetes/55-dra-p2-slice-class-claim/)
- [DRA P3: Workflow & Source](https://www.lixueduan.com/posts/kubernetes/57-dra-p3-workflow/)
- [DRA P4: Build Your Own Driver](https://www.lixueduan.com/posts/kubernetes/58-dra-p4-my-dra-driver/)

## License

MIT
