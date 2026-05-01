package env

import (
	"os"
	"strings"
)

// IsDockerContainer 判断当前环境是否为 Docker 容器
func IsDockerContainer() bool {
	// 方法一：检查 /.dockerenv 文件（最可靠，Docker 容器内必定存在）
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// 方法二：检查 /proc/1/cgroup
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "containerd") || strings.Contains(content, "kubepods") {
			return true
		}
	}

	// 方法三：检查 /proc/self/cgroup
	if data, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "containerd") || strings.Contains(content, "kubepods") {
			return true
		}
	}

	// 方法四：检查环境变量
	if os.Getenv("DOCKER_CONTAINER") != "" || os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	// 方法五：检查 /proc/1/comm
	if data, err := os.ReadFile("/proc/1/comm"); err == nil {
		content := strings.TrimSpace(string(data))
		if content == "docker-init" || content == "containerd-shim" || content == "runc" {
			return true
		}
	}

	// 注意：不再检查 /etc/hosts，因为 Docker Desktop for Mac/Windows 会在宿主机添加 docker 相关记录
	// 这会导致误判

	return false
}

// IsKubernetes 判断当前环境是否为 Kubernetes 集群
func IsKubernetes() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}
