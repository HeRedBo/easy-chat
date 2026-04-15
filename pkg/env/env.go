package env

import (
	"os"
	"strings"
)

// IsDockerContainer 判断当前环境是否为 Docker 容器
func IsDockerContainer() bool {
	// 方法一：检查 /proc/1/cgroup
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "containerd") || strings.Contains(content, "kubepods") {
			return true
		}
	}

	// 方法二：检查 /proc/self/cgroup
	if data, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "containerd") || strings.Contains(content, "kubepods") {
			return true
		}
	}

	// 方法三：检查环境变量
	if os.Getenv("DOCKER_CONTAINER") != "" || os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	// 方法四：检查 /proc/1/comm
	if data, err := os.ReadFile("/proc/1/comm"); err == nil {
		content := strings.TrimSpace(string(data))
		if content == "docker-init" || content == "containerd-shim" || content == "runc" {
			return true
		}
	}

	// 方法五：检查 /etc/hosts 文件
	if data, err := os.ReadFile("/etc/hosts"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "containerd") {
			return true
		}
	}

	// 方法六：检查是否存在 /.dockerenv 文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	return false
}

// IsKubernetes 判断当前环境是否为 Kubernetes 集群
func IsKubernetes() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}
