package env

import (
	"os"
	"testing"
)

func TestIsDockerContainer(t *testing.T) {
	// 在本地开发环境（Mac/Windows）应该返回 false
	// 除非你真的在 Docker 容器中运行测试
	
	result := IsDockerContainer()
	
	// 输出调试信息
	t.Logf("IsDockerContainer result: %v", result)
	
	// 如果你是在本地 Mac 上运行，应该为 false
	// 如果这个测试在 CI/CD 的 Docker 容器中运行，应该为 true
	// 这里不做断言，只是输出结果供人工验证
}

func TestIsDockerContainerWithEnv(t *testing.T) {
	// 测试环境变量检测
	originalValue := os.Getenv("DOCKER_CONTAINER")
	defer os.Setenv("DOCKER_CONTAINER", originalValue)
	
	// 设置环境变量
	os.Setenv("DOCKER_CONTAINER", "1")
	
	if !IsDockerContainer() {
		t.Error("Expected IsDockerContainer to return true when DOCKER_CONTAINER env is set")
	}
}

func TestIsKubernetes(t *testing.T) {
	originalValue := os.Getenv("KUBERNETES_SERVICE_HOST")
	defer os.Setenv("KUBERNETES_SERVICE_HOST", originalValue)
	
	// 默认应该为 false
	if IsKubernetes() {
		t.Log("Warning: KUBERNETES_SERVICE_HOST is set in current environment")
	}
	
	// 设置环境变量后应该为 true
	os.Setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
	if !IsKubernetes() {
		t.Error("Expected IsKubernetes to return true when KUBERNETES_SERVICE_HOST is set")
	}
}
