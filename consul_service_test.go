package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	// "strings"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/yunnysunny/consul-register/grpc_health_v1"
	"google.golang.org/grpc"
)

func TestInit(t *testing.T) {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		fmt.Println("读取网络地址失败", err)
		return
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				fmt.Println("得到ip地址：", ip)
				os.Setenv("SERVICE_ADDR", ip)
				return
			}

		}
	}

	fmt.Println("没有找到 ip 地址")
}

var httpHealthCalled = false
var grpcHealthCalled = false

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("http健康请求来")
	httpHealthCalled = true
	fmt.Fprintf(w, "ok")
}

type server struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (s *server) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	fmt.Println("grpc健康检查")
	grpcHealthCalled = true
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}
func (s *server) Watch(*grpc_health_v1.HealthCheckRequest, grpc_health_v1.Health_WatchServer) error {
	return nil
}

func TestConsulService(t *testing.T) {
	const (
		httpServicePort       = 8003
		httpServiceCheckPath  = "/health"
		httpServiceNameHealth = "test-http-health"
		grpcServicePort       = 8004
		grpcServiceNameHealth = "test-grpc-health"
	)

	func() {
		http.HandleFunc(httpServiceCheckPath, indexHandler)
		go func() {
			err := http.ListenAndServe(fmt.Sprintf(":%d", httpServicePort), nil)
			if err != nil {
				fmt.Println("启动http失败", err)
			}
		}()
	}()

	func() {
		address := ":" + strconv.Itoa(grpcServicePort)
		listen, err := net.Listen("tcp", address)
		if err != nil {
			fmt.Printf("grpc 开启端口失败%s\n", err)
			t.Fatal()
			return
		}
		s := grpc.NewServer()
		grpc_health_v1.RegisterHealthServer(s, &server{})
		fmt.Println("grpc 启动服务成功", listen.Addr().String())
		go s.Serve(listen)
	}()

	config := api.DefaultConfig()
	config.Address = os.Getenv("CONSUL_ADDR")
	client, err := api.NewClient(config)
	if err != nil {
		fmt.Println("创建 consul 客服端失败", err)
		t.Fatal()
		return
	}

	var getServiceCount = func(serviceName string, tagName string) (int, map[string]string) {
		services, _, err := client.Health().Service(serviceName, tagName, true, nil)
		if err != nil {
			fmt.Printf("error retrieving instances from Consul: %v", err)
			return 0, nil
		}
		count := len(services)
		var meta map[string]string
		if count > 0 {
			meta = services[0].Service.Meta
		}
		return count, meta
	}

	var cleanEnv = func() {
		os.Unsetenv("EXPOSED_TO_GATE")
	}
	var mySleep = func() {
		time.Sleep(time.Second * (SERVICE_CHECK_INTERVAL_SECONDS + 2))
		fmt.Println("sleep完成")
	}

	Convey("http健康检查", t, func(c C) {
		option := &ConsulServiceOption{
			ServiceName:          httpServiceNameHealth,
			ServicePort:          httpServicePort,
			ServiceType:     SERVICE_TYPE_HTTP,
			ServiceCheckHttpPath: httpServiceCheckPath,
		}
		service, err := NewInstanceFromEnv(option)
		So(err, ShouldEqual, nil)
		result := <-service.RegistrationResult
		So(result, ShouldEqual, REGISTRATION_STATUS_SUCCESS)
		mySleep()

		So(httpHealthCalled, ShouldBeTrue)
		err = service.Deregister()
		So(err, ShouldBeNil)
	})

	Convey("grpc健康检查", t, func(c C) {

		option := &ConsulServiceOption{
			ServiceName:      grpcServiceNameHealth,
			ServicePort:      grpcServicePort,
			ServiceType: SERVICE_TYPE_GRPC,
		}
		service, err := NewInstanceFromEnv(option)
		So(err, ShouldEqual, nil)
		result := <-service.RegistrationResult
		So(result, ShouldEqual, REGISTRATION_STATUS_SUCCESS)
		mySleep()

		So(grpcHealthCalled, ShouldBeTrue)
		err = service.Deregister()
		So(err, ShouldBeNil)
	})

	Convey("http使用EXPOSED_TO_GATE", t, func(c C) {
		cleanEnv()
		const IDC = "idc"
		os.Setenv("EXPOSED_TO_GATE", "true")
		os.Setenv("CLUSTER_TYPE", IDC)
		option := &ConsulServiceOption{
			ServiceName:          httpServiceNameHealth,
			ServicePort:          httpServicePort,
			ServiceType:     SERVICE_TYPE_HTTP,
			ServiceCheckHttpPath: httpServiceCheckPath,
		}
		service, err := NewInstanceFromEnv(option)
		So(err, ShouldEqual, nil)
		result := <-service.RegistrationResult
		So(result, ShouldEqual, REGISTRATION_STATUS_SUCCESS)
		mySleep() //SLEEP
		count, meta := getServiceCount(httpServiceNameHealth, TAG_NAME_EXPOSED_TO_GATE)
		So(count, ShouldBeGreaterThan, 0)
		So(meta["ServiceName"], ShouldEqual, httpServiceNameHealth)
		So(meta["ServicePath"], ShouldEqual, "/"+IDC)
		So(meta["ClusterType"], ShouldEqual, IDC)

		err = service.Deregister()
		So(err, ShouldBeNil)
	})
}
