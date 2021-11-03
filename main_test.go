package main

import (
	"fmt"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func unsetEnv() {
	os.Unsetenv(ENV_REGISTERED_HTTP_SERVICE_NAME)
	os.Unsetenv(ENV_REGISTERED_GRPC_SERVICE_NAME)
}

func TestMainEntry(t *testing.T) {
	const (
		httpPort = 8000
		httpCheckPath = "/health"
		httpServiceName = "test-http-main"
		grpcPort = 8001
		grpcServiceName = "test-grpc-main"
	)
	Convey("http注册", t, func(c C) {
		unsetEnv()

		os.Setenv(ENV_REGISTERED_HTTP_SERVICE_NAME, httpServiceName)
		os.Setenv(ENV_REGISTERED_HTTP_SERVICE_PORT, fmt.Sprintf("%d", httpPort))
		os.Setenv(ENV_REGISTERED_HTTP_HEALTH_CHECK_PATH, httpCheckPath)
		main()
	})

	Convey("grpc注册", t, func(c C) {
		unsetEnv()

		os.Setenv(ENV_REGISTERED_GRPC_SERVICE_NAME, grpcServiceName)
		os.Setenv(ENV_REGISTERED_GRPC_SERVICE_PORT, fmt.Sprintf("%d", grpcPort))
		main()
	})
}