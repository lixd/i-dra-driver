package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lixd/i-dra-driver/pkg/cdi"
	"github.com/lixd/i-dra-driver/pkg/common"
	"github.com/lixd/i-dra-driver/pkg/device"
	"github.com/lixd/i-dra-driver/pkg/plugin"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/dynamic-resource-allocation/kubeletplugin"
)

func main() {
	var nodeName string
	var devicePath string
	var driverName string
	var rescanInterval time.Duration

	flag.StringVar(&nodeName, "node-name", "", "Name of the node this driver runs on (required, usually from NODE_NAME env)")
	flag.StringVar(&devicePath, "device-path", common.DevicePath, "Path to the directory containing gopher device files")
	flag.StringVar(&driverName, "driver-name", common.DriverName, "DRA driver name")
	flag.DurationVar(&rescanInterval, "rescan-interval", common.RescanInterval, "Interval for periodic device rescan")
	klog.InitFlags(nil)
	flag.Parse()

	if nodeName == "" {
		nodeName = os.Getenv("NODE_NAME")
	}
	if nodeName == "" {
		klog.Fatalf("--node-name or NODE_NAME environment variable is required")
	}

	klog.Infof("starting i-dra-driver: node=%s driver=%s device-path=%s rescan=%s",
		nodeName, driverName, devicePath, rescanInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		klog.Infof("received signal %v, shutting down", sig)
		cancel()
	}()

	cfg, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("get in-cluster config failed: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("create kubernetes client failed: %v", err)
	}

	cdiHandler, err := cdi.NewHandler(driverName, common.CDIVendor, common.CDIClass, devicePath)
	if err != nil {
		klog.Fatalf("create CDI handler failed: %v", err)
	}

	draPlugin := plugin.NewGopherDRAPlugin(cdiHandler)

	helper, err := kubeletplugin.Start(
		ctx,
		draPlugin,
		kubeletplugin.DriverName(driverName),
		kubeletplugin.KubeClient(kubeClient),
		kubeletplugin.NodeName(nodeName),
	)
	if err != nil {
		klog.Fatalf("start DRA plugin helper failed: %v", err)
	}
	defer helper.Stop()

	devices, err := device.Discover(devicePath)
	if err != nil {
		klog.Fatalf("initial device discovery failed: %v", err)
	}

	resources := device.BuildDriverResources(nodeName, devices)
	if err := helper.PublishResources(ctx, resources); err != nil {
		klog.Fatalf("publish initial ResourceSlice failed: %v", err)
	}
	klog.Infof("published initial ResourceSlice with %d devices", len(devices))

	go rescanLoop(ctx, helper, nodeName, devicePath, rescanInterval)

	klog.Infof("i-dra-driver started successfully")
	<-ctx.Done()
	klog.Infof("i-dra-driver shutting down")
}

func rescanLoop(ctx context.Context, helper *kubeletplugin.Helper, nodeName, devicePath string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			devices, err := device.Discover(devicePath)
			if err != nil {
				klog.Errorf("periodic device discovery failed: %v", err)
				continue
			}

			resources := device.BuildDriverResources(nodeName, devices)
			if err := helper.PublishResources(ctx, resources); err != nil {
				klog.Errorf("publish ResourceSlice failed: %v", err)
			} else {
				klog.Infof("republished ResourceSlice with %d devices", len(devices))
			}
		}
	}
}
