# consul-register

[![codecov](https://codecov.io/gh/yunnysunny/consul-register/branch/main/graph/badge.svg?token=2JQ40ZUNF2)](https://codecov.io/gh/yunnysunny/consul-register)

consul 服务注册插件，在系统启动时根据环境变量注册所需的服务到 consul 服务节点

## 使用

### 环境变量说明

程序启动后会自动读取环境变量，只要存在 `REGISTERED_HTTP_SERVICE_NAME` 就注册 http 服务，如果存在 `REGISTERED_GRPC_SERVICE_NAME` 就注册 grpc 服务。

注册 http 服务时，会读取如下环境变量

| 环境变量                              | 作用         |      |
| ------------------------------------- | ------------ | ---- |
| REGISTERED_HTTP_SERVICE_NAME          | 服务名称     | 必选 |
| REGISTERED_HTTP_SERVICE_PORT          | 服务端口     | 必选 |
| REGISTERED_HTTP_HEALTH_CHECK_PATH | 健康检查路径 | 必选 |

注册 grpc 服务时，会读取如下环境变量

| 环境变量                         | 作用     |      |
| -------------------------------- | -------- | ---- |
| REGISTERED_GRPC_SERVICE_NAME | 服务名称 | 必选 |
| REGISTERED_GRPC_SERVICE_PORT | 服务端口 | 必选 |
|                                  |          |      |

不管是 http 还是 grpc，两者通用的环境变量如下

| 环境变量                                  | 作用                                                         |                                     |
| ----------------------------------------- | ------------------------------------------------------------ | ----------------------------------- |
| CONSUL_ADDR                               | consul 的 client 端地址                                      | 必选¹                               |
| CLUSTER_TYPE                              | 机房类型                                                     | 必选¹                               |
| CLUSTER_ID                                | 机房标识                                                     | 必选²                               |
| RETRY_REGISTER_DELAY_MS                   | 注册失败后的重试时间间隔                                     | 默认为3s                            |
| RETRY_REGISTER_MAX_TIMES                  | 注册失败后的重试次数                                         | 默认为300                           |
| EXPOSED_TO_GATE                           | 是否注册为机房相关应用                                       | 可选                                |
| DEREGISTER_CRITICAL_SERVICE_AFTER_SECONDS | 健康检查失败后多久后，取消注册                               | 默认为0，代表检查检查失败不取消注册 |
| SERVICE_ADDR                              | 对于应用的可访问 ip，默认是读取系统网卡来获取，但是也可以通过这个环境变量来手动指定 | 可选                                |

注解1：必须存在的环境变量

注解2：可选环境变量



### 自定义编译
使用自定义编译，可以使用最新代码来生成可执行程序。

```shell
git clone git@github.com:yunnysunny/consul-register.git
cd consul-register
go mod tidy
go build -o ./bin/consul-register && chmod +x ./bin/consul-register
```

### 下载安装
使用 go install 可以选择安装最新稳定版本生成的可执行程序。

```shell
go install github.com:yunnysunny/consul-register@v0.1.0
```

安装完之后会在 $GOPATH/bin 目录下生成可执行文件 `consul-register`。

## 测试

本地测试依赖于 环境变量 `CONSUL_ADDR`，同时为了避免产生脏数据，也相应的设置上 `DEREGISTER_CRITICAL_SERVICE_AFTER_SECONDS` 为一个确定的数值，比如说 `5`。正式使用的时候，`DEREGISTER_CRITICAL_SERVICE_AFTER_SECONDS` 不需要设置。
