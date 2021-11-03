package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
)
type ServiceType int
const (
	SERVICE_TYPE_NONE ServiceType = iota
	SERVICE_TYPE_HTTP
	SERVICE_TYPE_GRPC
)
type RegistrationStatus int

const (
	REGISTRATION_STATUS_INIT RegistrationStatus = iota
	REGISTRATION_STATUS_SUCCESS
	REGISTRATION_STATUS_HAS_DEREGISTER
	REGISTRATION_STATUS_EXCEED_FAILED_TIMES

)

/**
 * @constant {String} TAG_NAME_EXPOSED_TO_GATE 是否被前置 nginx-cahce 集群读取的 tag 名称，其值为 `Used4Nginx`。
 */
 const TAG_NAME_EXPOSED_TO_GATE = "EXPOSED_TO_GATE"


 const SERVICE_CHECK_INTERVAL_SECONDS = 6

 const RETRY_TIMES_AFTER_FAILED = 300

 const RETRY_DELAY_TIME_AFTER_FAILED = 3000

type ConsulServiceOption struct {
	ServiceName string
	ServicePort int
	ServiceAddr string
	ServiceType ServiceType
	ClusterId string
	ClusterType string
	IsExposedToGate bool
	RetryRegisterIntervalMs uint32
	RetryRegisterMaxTimes uint32
	ServiceMeta map[string]string
	ServiceTags []string
	DeRegisterCriticalServiceAfterSeconds uint32
	ServiceCheckIntervalSeconds uint32
	ServiceCheckHttpPath string

}

type ConsulService struct {
	option *ConsulServiceOption
	client *api.Client
	hasDeregister bool
	serviceId string
	RegistrationResult chan RegistrationStatus
	RegistrationErrorMsg string
	RegistrationStatus RegistrationStatus
	retryTimes uint32
}

func getClientIp() (string ,error) {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		return "", err
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}

		}
	}

	return "", errors.New("can not find the client ip address")

}

func NewInstance(consulAddr string, option *ConsulServiceOption) (*ConsulService, error) {
	config := api.DefaultConfig()
	config.Address = consulAddr
	client, err := api.NewClient(config)
	if err != nil {
		fmt.Printf("error create consul client: %v\n", err)
		return nil, err
	}
	if option.RetryRegisterMaxTimes == 0 {
		option.RetryRegisterMaxTimes = RETRY_TIMES_AFTER_FAILED
	}
	if option.RetryRegisterIntervalMs == 0 {
		option.RetryRegisterIntervalMs = RETRY_DELAY_TIME_AFTER_FAILED
	}
	consulService := &ConsulService {
		option: option,
		client: client,
		RegistrationResult: make(chan RegistrationStatus, 1),
	}
	consulService.register()
	return consulService, nil
}

func NewInstanceFromEnv(option *ConsulServiceOption) (*ConsulService, error) {
	clusterType := os.Getenv("CLUSTER_TYPE")
	if clusterType != "" {
		option.ClusterType = clusterType
	}
	clusterId := os.Getenv("CLUSTER_ID")
	if clusterId != "" {
		option.ClusterId = clusterId
	}
	retryRegisterIntervalMs, _ := strconv.ParseUint(os.Getenv("RETRY_REGISTER_DELAY_MS"), 10, 32)
	if retryRegisterIntervalMs > 0 {
		option.RetryRegisterIntervalMs = uint32(retryRegisterIntervalMs)
	}

	retryRegisterMaxTimes, _ := strconv.ParseUint(os.Getenv("RETRY_REGISTER_MAX_TIMES"), 10, 32)
	if retryRegisterMaxTimes > 0 {
		option.RetryRegisterMaxTimes = uint32(retryRegisterMaxTimes)
	}
	deRegisterCriticalServcieAfterSeconds,_ := 
		strconv.ParseUint(os.Getenv("DEREGISTER_CRITICAL_SERVICE_AFTER_SECONDS"), 10, 32)
	if deRegisterCriticalServcieAfterSeconds > 0 {
		option.DeRegisterCriticalServiceAfterSeconds = uint32(deRegisterCriticalServcieAfterSeconds)
	}
	option.IsExposedToGate = os.Getenv("EXPOSED_TO_GATE") == "true"

	return NewInstance(os.Getenv("CONSUL_ADDR"), option)
}

func (consulService *ConsulService) register() {
	option := consulService.option
	serviceName := option.ServiceName
	if consulService.hasDeregister {
		fmt.Printf("服务%s已经取消注册", serviceName)
		return
	}
	
	
	clusterId := option.ClusterId
	servicePath := "/" + option.ClusterType
	if clusterId != "" {
		servicePath = servicePath + "/" + clusterId
	}
	
	tags := option.ServiceTags
	if tags == nil {
		tags = make([]string, 0)
	}
	if option.IsExposedToGate {
		tags = append(tags, TAG_NAME_EXPOSED_TO_GATE)
	}
	serviceMeta := option.ServiceMeta
	if serviceMeta == nil {
		serviceMeta = map[string]string{}
	}
	serviceMeta["ServiceName"] = serviceName
	serviceMeta["ServicePath"] = servicePath
	serviceMeta["ClusterType"] = option.ClusterType

	ip, _ := getClientIp()
	ipFromEnv := os.Getenv("SERVICE_ADDR")
	if ipFromEnv != "" {
		ip = ipFromEnv
	}
	// 创建注册到consul的服务到
	registration := new(api.AgentServiceRegistration)
	registration.ID = serviceName + "-" + ip
	consulService.serviceId = registration.ID
	registration.Name = serviceName
	registration.Port = option.ServicePort
	registration.Tags = tags
	registration.Address = ip
	registration.Meta = serviceMeta

	// 增加consul健康检查回调函数	
	check := new(api.AgentServiceCheck)
	if option.ServiceType == SERVICE_TYPE_HTTP {
		check.HTTP = "http://" + ip + ":" + strconv.Itoa(option.ServicePort) + option.ServiceCheckHttpPath
	} else if option.ServiceType == SERVICE_TYPE_GRPC {
		check.GRPC = ip + ":" + strconv.Itoa(option.ServicePort) + "/" + serviceName
	} else {
		fmt.Println("非法的服务类型："+ strconv.Itoa(int(option.ServiceType)))
		return 
	}
	checkTTL := option.ServiceCheckIntervalSeconds
	if checkTTL == 0 {
		checkTTL= SERVICE_CHECK_INTERVAL_SECONDS
	}
	check.Interval = strconv.Itoa(int(checkTTL)) + "s"
	check.Timeout = check.Interval
	if option.DeRegisterCriticalServiceAfterSeconds > 0 {
		check.DeregisterCriticalServiceAfter = strconv.Itoa(int(option.DeRegisterCriticalServiceAfterSeconds)) + "s"
	}
	registration.Check = check

	go func() {
		for  {
			if (consulService.hasDeregister) {
				fmt.Printf("服务%s已经取消注册\n", serviceName)

				consulService.genResult(REGISTRATION_STATUS_HAS_DEREGISTER, "服务已经取消注册")
				break
			}
			fmt.Println("开始注册",registration.ID)
			err := consulService.client.Agent().ServiceRegister(registration)
			if err != nil {
				fmt.Printf("注册%s服务失败%s\n", serviceName, err)
				consulService.retryTimes++
				if consulService.retryTimes >= (option.RetryRegisterMaxTimes) {
					fmt.Printf("注册%s服务失败超过次数限制%d\n", serviceName, option.RetryRegisterMaxTimes)
					consulService.genResult(REGISTRATION_STATUS_EXCEED_FAILED_TIMES, "注册服务失败超过次数限制")
					break
				} else {
					time.Sleep(time.Millisecond * time.Duration(option.RetryRegisterIntervalMs))
				}				
			} else {
				fmt.Printf("注册%s服务成功\n", serviceName)
				consulService.genResult(REGISTRATION_STATUS_SUCCESS, "")
				break
			}
		}		
	}()
}

func (consulService *ConsulService) genResult(statusCode RegistrationStatus, errMsg string) {
	consulService.RegistrationStatus = statusCode
	consulService.RegistrationErrorMsg = errMsg
	consulService.RegistrationResult <- statusCode
}

func (consulService *ConsulService) Deregister() (error) {
	if consulService.serviceId == "" {
		fmt.Printf("服务尚未开始注册\n")
		return errors.New("服务尚未开始注册")
	}
	consulService.hasDeregister = true
	//直接取消注册
	err := consulService.client.Agent().ServiceDeregister(consulService.serviceId)
	if err != nil {
		fmt.Printf("取消%s注册服务失败:%s\n", consulService.option.ServiceName, err)
	} else {
		fmt.Printf("取消%s服务注册成功\n", consulService.option.ServiceName)
	}
	return err
}
