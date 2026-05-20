# i-dra-driver

[English](README_en.md)

自定义 Kubernetes DRA (Dynamic Resource Allocation) 驱动，将文件作为 "gopher" 设备暴露给 Pod 使用。

这是博客 [从零开发自己的 DRA 驱动](https://www.lixueduan.com/posts/kubernetes/58-dra-p4-my-dra-driver/) 的配套项目，用于与传统的 [DevicePlugin](https://www.lixueduan.com/posts/kubernetes/21-device-plugin/) 方案进行对比。

不熟悉 DRA 的同学可以先看下这几篇文章：
- [DRA P1：DRA 能解决什么问题？从部署到使用的完整体验](https://www.lixueduan.com/posts/kubernetes/54-dra-p1-quickstart/)
- [DRA P2：理解 DRA：ResourceSlice、Claim、Class 三角关系](https://www.lixueduan.com/posts/kubernetes/55-dra-p2-slice-class-claim/)
- [DRA P3：DRA 工作流程与源码分析](https://www.lixueduan.com/posts/kubernetes/57-dra-p3-workflow/)
- [DRA P4：从零开发自己的 DRA 驱动](https://www.lixueduan.com/posts/kubernetes/58-dra-p4-my-dra-driver/)

### 微信公众号：探索云原生

一个云原生打工人的探索之路，专注云原生，Go，坚持分享最佳实践、经验干货。

扫描下面二维码，关注我即时获取更新~

![](https://img.lixueduan.com/about/wechat/qrcode_search.png)

## 工作原理

驱动扫描节点上 `/etc/gophers` 目录下的文件，将每个文件作为一个 "gopher" 设备通过 DRA 框架暴露：

- **设备发现**：扫描 `/etc/gophers`，每个文件生成一个设备，包含 `type=gopher` 和 `size=<文件大小>` 属性
- **ResourceSlice**：将设备清单（含属性和容量）发布到 API Server，调度器可见
- **DeviceClass**：通过 CEL 选择器匹配 `device.driver == 'gopher.example.com' && device.attributes['gopher.example.com'].type == 'gopher'`
- **CDI 注入**：在 `NodePrepareResources` 阶段创建 CDI spec，向容器注入 `GOPHER=<设备名>` 环境变量

## 前置条件

- Kubernetes 1.32+ 且启用 DRA feature gate（v1.34+ 已默认启用）
- Container Runtime 启用 CDI（containerd v1.7+ 默认启用）

## 快速开始

### 1. 在节点上准备设备文件

```bash
mkdir -p /etc/gophers
echo "hello from gopher-a" > /etc/gophers/gopher-a
echo "hello from gopher-b" > /etc/gophers/gopher-b
```

### 2. 部署驱动

```bash
# RBAC
kubectl apply -f deploy/rbac.yaml

# DaemonSet
kubectl apply -f deploy/daemonset.yaml

# DeviceClass
kubectl apply -f deploy/deviceclass.yaml
```

> **注意**：DaemonSet 启动前，宿主机上需先创建 plugin socket 目录：`mkdir -p /var/lib/kubelet/plugins/gopher.example.com/`

### 3. 验证 ResourceSlice

```bash
$ kubectl get resourceslice | grep gopher
NAME                                        NODE         DRIVER                  POOL         AGE
00000-gopher.example.com-ecs-a10-sh-nj7lm   ecs-a10-sh   gopher.example.com      ecs-a10-sh   18s
```

查看设备详情：

```bash
kubectl get resourceslice <name> -oyaml
```

可以看到每个设备的属性（type=gopher）和容量（size=20），而 DevicePlugin 只能在 Node capacity 上显示数量（如 `lixueduan.com/gopher: "2"`）。

### 4. 运行测试 Pod

```bash
kubectl apply -f deploy/test-pod.yaml
```

验证：

```bash
# Pod 运行正常
$ kubectl get pod gopher-test-pod
NAME              READY   STATUS    RESTARTS   AGE
gopher-test-pod   1/1     Running   0          15s

# 查看设备注入结果
$ kubectl logs gopher-test-pod
GOPHER env: gopher-a
Gopher device allocated successfully!
Device file: /etc/gophers/gopher-a
hello from gopher-a

# 验证容器内环境变量
$ kubectl exec gopher-test-pod -- env | grep GOPHER
GOPHER=gopher-a
```

### 5. 查看 ResourceClaim 分配结果

```bash
$ kubectl get resourceclaim
NAME                                 STATE                AGE
gopher-test-pod-gopher-claim-9chj8   allocated,reserved   28s
```

状态为 `allocated,reserved`，说明调度器已成功分配设备并预留。

## 构建镜像

```bash
# 编译二进制
make build

# 构建 Docker 镜像
make build-image
```

## 项目结构

```
i-dra-driver/
  cmd/main.go                 # 入口
  pkg/
    common/constant.go        # 常量（驱动名、设备路径、CDI vendor）
    device/
      discovery.go            # 设备发现：扫描 /etc/gophers
      resourceslice.go        # 构建 DriverResources
    cdi/handler.go            # CDI spec 管理：创建/删除
    plugin/plugin.go          # DRAPlugin 接口实现
  deploy/
    rbac.yaml                 # ServiceAccount + ClusterRole + ClusterRoleBinding
    daemonset.yaml            # DaemonSet 部署
    deviceclass.yaml          # DeviceClass（CEL 选择器）
    test-pod.yaml             # ResourceClaimTemplate + 测试 Pod
```

## 相关项目

- [i-device-plugin](https://github.com/lixd/i-device-plugin) — DevicePlugin 版本的实现
- [DRA P1: 部署实战](https://www.lixueduan.com/posts/kubernetes/54-dra-p1-quickstart/)
- [DRA P2: Slice、Class、Claim](https://www.lixueduan.com/posts/kubernetes/55-dra-p2-slice-class-claim/)
- [DRA P3: 工作流程与源码分析](https://www.lixueduan.com/posts/kubernetes/57-dra-p3-workflow/)
- [DRA P4: 从零开发自己的 DRA 驱动](https://www.lixueduan.com/posts/kubernetes/58-dra-p4-my-dra-driver/)

## License

MIT
