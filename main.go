package main

import (
	"fmt"
	"os"
	"strconv"
)

const (
	ENV_REGISTERED_HTTP_SERVICE_NAME = "REGISTERED_HTTP_SERVICE_NAME"
	ENV_REGISTERED_HTTP_SERVICE_PORT = "REGISTERED_HTTP_SERVICE_PORT"
	ENV_REGISTERED_HTTP_HEALTH_CHECK_PATH = "REGISTERED_HTTP_HEALTH_CHECK_PATH"
	ENV_REGISTERED_GRPC_SERVICE_NAME = "REGISTERED_GRPC_SERVICE_NAME"
	ENV_REGISTERED_GRPC_SERVICE_PORT = "REGISTERED_GRPC_SERVICE_PORT"
)

func main() {
	consulAddr := os.Getenv("CONSUL_ADDR")
	if consulAddr == "" {
		fmt.Println("consul地址环境变量不存在")
		os.Exit(120)
		return
	}
	fmt.Println("使用consul地址", consulAddr)
	var chans  [] chan RegistrationStatus 
	httpServiceName := os.Getenv(ENV_REGISTERED_HTTP_SERVICE_NAME)
	if httpServiceName != "" {
		port, _ := strconv.ParseInt(os.Getenv(ENV_REGISTERED_HTTP_SERVICE_PORT), 10 ,32)
		option := &ConsulServiceOption {
			ServiceName: httpServiceName,
			ServicePort: int(port),
			ServiceType: SERVICE_TYPE_HTTP,
			ServiceCheckHttpPath: os.Getenv(ENV_REGISTERED_HTTP_HEALTH_CHECK_PATH),
		}
		service, err := NewInstanceFromEnv(option)
		if err == nil {
			chans = append(chans, service.RegistrationResult)
		}
	}
	grpcServiceName := os.Getenv(ENV_REGISTERED_GRPC_SERVICE_NAME)
	if grpcServiceName != "" {
		port, _ := strconv.ParseInt(os.Getenv(ENV_REGISTERED_GRPC_SERVICE_PORT), 10, 32)
		option := &ConsulServiceOption{
			ServiceName: grpcServiceName,
			ServicePort: int(port),
			ServiceType: SERVICE_TYPE_GRPC,
		}
		service, err := NewInstanceFromEnv(option)
		if err == nil {
			chans = append(chans, service.RegistrationResult)
		}
	}
	for _, ch := range chans {
		status := <-ch
		if status != REGISTRATION_STATUS_SUCCESS {
			os.Exit(125)//如果注册不成功，就强制退出
		}
	}
}