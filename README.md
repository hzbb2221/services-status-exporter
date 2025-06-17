# Services Status Exporter

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-Available-2496ED?logo=docker)](https://hub.docker.com/)

一个轻量级的服务状态监控工具，专为微服务架构设计。通过HTTP请求监控关键业务服务的可用性和响应性能，暴露Prometheus兼容的指标，可与Grafana等可视化工具无缝集成。

## 🚀 快速开始

### 方式一：Docker Compose（推荐）
```bash
# 使用docker-compose启动（推荐方式）
docker-compose up -d
```

### 方式二：Docker
```bash
# 使用Docker运行
docker run -d \
  --name services-status-exporter \
  -p 9112:9112 \
  -v $(pwd)/web:/app/web \
  services-status-exporter
```

### 方式三：直接运行
```bash
# 下载并运行
./services-status-exporter
```

## 项目背景
本工具用于监控关键业务网站/服务的可用性和响应性能，通过暴露Prometheus兼容的指标，可与Grafana等可视化工具集成，实现服务状态的实时监控与告警。适用于微服务架构中对第三方依赖服务的健康检查场景。

## ✨ 功能特点

- **🔧 多配置支持**：支持从`web`目录自动加载所有YAML配置文件（支持`.yaml`和`.yml`后缀）
- **🌐 灵活请求**：支持GET/POST等HTTP方法（根据是否配置`body`自动判断），可自定义请求头/请求体
- **📊 精准监控**：提供响应时间（直方图）和请求状态（计数器）双维度指标，支持按站点/状态码分组
- **🚀 轻量部署**：支持二进制直接运行、Docker容器化及Docker Compose编排三种部署方式
- **🔒 安全认证**：支持TLS加密和Basic Auth基本认证
- **⚡ 高性能**：Go语言编写，内存占用低，并发性能优异
- **📈 Prometheus兼容**：完全兼容Prometheus指标格式，可直接集成到现有监控体系

## 监控指标详解
| 指标名称                     | 类型       | 说明                         | 标签（Labels）               |
|------------------------------|------------|------------------------------|------------------------------|
| `site_request_duration_seconds` | Gauge      | HTTP请求响应时间（单位：秒） | `site`（站点URL）、`method`（请求方法）、`env`（环境）、`service`（服务名）、`type`（类型） |
| `site_status_code`           | Gauge      | HTTP响应状态码               | `site`（站点URL）、`method`（请求方法）、`env`（环境）、`service`（服务名）、`type`（类型） |
| `site_request_total`          | Counter    | 总请求数（按请求状态分类）     | `site`（站点URL）、`method`（请求方法）、`status`（请求状态：success/error）、`env`（环境）、`service`（服务名）、`type`（类型） |
| `site_up`                    | Gauge      | 站点可用性状态（1=可用，0=不可用） | `site`（站点URL）、`method`（请求方法）、`env`（环境）、`service`（服务名）、`type`（类型） |
| `services_status_exporter_info` | Gauge   | 应用程序信息（版本、构建时间等） | `version`（版本号）、`build_time`（构建时间） |
| `services_status_exporter_up`   | Gauge   | 应用程序运行状态（1=运行中，0=停止） | 无 |

## 📋 配置说明

### 命令行参数

```bash
./services-status-exporter [选项]

选项：
  -port int
        HTTP服务端口 (默认: 9112)
  -web-dir string
        配置文件目录 (默认: "web")
  -web-config string
        Web服务器配置文件路径 (可选)
  -version
        显示版本信息
  -help
        显示帮助信息
```

### 监控配置文件规范

- 配置文件需放置在`web`目录（可通过`-web-dir`参数修改）
- 单个文件对应一个监控站点，建议以`[站点名].yaml`格式命名（如`api-service.yaml`）
- 支持同时存在多个配置文件，程序启动时会自动加载所有`.yaml`和`.yml`文件

### 监控配置参数详解

| 参数名称 | 类型 | 必填 | 默认值 | 说明 |
|----------|------|------|--------|---------|
| `url` | string | ✅ | - | 监控的目标URL |
| `method` | string | ❌ | GET | HTTP请求方法（GET/POST/PUT等） |
| `headers` | map | ❌ | {} | 自定义请求头 |
| `body` | string | ❌ | "" | 请求体内容（存在时自动设置为POST） |
| `interval_seconds` | int | ❌ | 30 | 监控间隔时间（秒） |
| `timeout_seconds` | int | ❌ | 10 | 请求超时时间（秒） |
| `skip_verify` | bool | ❌ | false | 是否跳过HTTPS证书验证 |
| `labels` | map | ❌ | {} | 自定义标签，支持字段：`env`（环境）、`service`（服务名）、`type`（类型） |

### 配置示例

#### 1. 简单GET请求监控
```yaml
# web/website.yaml
url: "https://example.com"
interval_seconds: 60
timeout_seconds: 5
labels:
  service: "官网"
  env: "生产环境"
  type: "web"
```

#### 2. 带认证的API监控
```yaml
# web/api-service.yaml
url: "https://api.example.com/health"
headers:
  Authorization: "Bearer your-token-here"
  User-Agent: "Services-Status-Exporter/1.0"
interval_seconds: 30
timeout_seconds: 10
labels:
  service: "用户API"
  env: "生产环境"
  type: "api"
```

#### 3. POST请求监控
```yaml
# web/auth-api.yaml
url: "https://auth.example.com/login"
method: POST
headers:
  Content-Type: "application/json"
  X-API-Key: "your-api-key"
body: '{"username": "monitor", "password": "secret"}'
interval_seconds: 120
timeout_seconds: 15
labels:
  service: "认证服务"
  env: "测试环境"
  type: "api"
```

#### 4. 跳过SSL证书验证监控
```yaml
# web/internal-service.yaml
url: "https://internal.example.com/health"
skip_verify: true
interval_seconds: 30
timeout_seconds: 10
labels:
  service: "内部服务"
  env: "开发环境"
  type: "internal"
```

### Web服务器配置（可选）

如需启用TLS或基本认证，可创建`web-config.yml`文件：

```yaml
# TLS配置
tls_server_config:
  cert_file: server.crt
  key_file: server.key

# 基本认证配置
basic_auth_users:
  admin: $2y$12$JlRtb8Gb/W0I0cp6VCdztuEOIvbKAJ6C6jXIt1h.PTaUoNrHiIBa2  # 密码: admin
```

然后使用`-web-config`参数启动：
```bash
./services-status-exporter -web.config.file=web-config.yml
```

## 🔧 使用说明

### 1. 启动服务

```bash
# 基本启动
./services-status-exporter

# 自定义端口和配置目录
./services-status-exporter -addr=:19115 -web-dir=./web

# 启用TLS和认证
./services-status-exporter -web.config.file=web-config.yml
```

### 2. 访问指标

启动后，可通过以下端点访问：

- **指标端点**: `http://localhost:9112/metrics` - Prometheus指标

> **注意**: 默认监听端口为 `9112`，可通过 `-addr` 参数自定义

### 3. 指标示例

```prometheus
# 响应时间（秒）
site_request_duration_seconds{site="https://example.com",method="GET",env="生产环境",service="示例服务",type="web"} 0.245

# HTTP响应状态码
site_status_code{site="https://example.com",method="GET",env="生产环境",service="示例服务",type="web"} 200

# 请求总数计数器
site_request_total{site="https://example.com",method="GET",status="success",env="生产环境",service="示例服务",type="web"} 48
site_request_total{site="https://example.com",method="GET",status="error",env="生产环境",service="示例服务",type="web"} 2

# 站点可用性状态
site_up{site="https://example.com",method="GET",env="生产环境",service="示例服务",type="web"} 1

# 应用程序信息和运行状态
services_status_exporter_info{version="1.0.0",build_time="2024-01-15T10:30:00Z"} 1
services_status_exporter_up 1
```

## 🐳 部署指南

> **推荐使用 Docker Compose 部署**，简单快捷，无需额外配置 Prometheus

### Docker Compose部署（推荐）

```bash
# 克隆项目
git clone <repository-url>
cd services-status-exporter

# 运行
docker-compose up -d
```

### Docker部署

#### 1. 构建镜像
```bash
# 克隆项目
git clone <repository-url>
cd services-status-exporter

# 构建Docker镜像
docker build -t services-status-exporter .
```

#### 2. 运行容器
```bash
# 基本运行
docker run -d \
  --name status-exporter \
  -p 9112:9112 \
  -v $(pwd)/web:/app/web \
  services-status-exporter
```

## 📊 Prometheus集成

### Prometheus配置

在`prometheus.yml`中添加以下配置：

#### 基础配置
```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'services-status-exporter'
    static_configs:
      - targets: ['localhost:9112']
    scrape_interval: 30s
    metrics_path: /metrics
```

#### 仅SSL配置（跳过证书验证）
```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'services-status-exporter-ssl'
    static_configs:
      - targets: ['your-domain.com:9112']
    scrape_interval: 30s
    metrics_path: /metrics
    scheme: https
    tls_config:
      insecure_skip_verify: true
```

#### 仅Basic Auth配置
```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'services-status-exporter-auth'
    static_configs:
      - targets: ['localhost:9112']
    scrape_interval: 30s
    metrics_path: /metrics
    basic_auth:
      username: 'admin'
      password: 'your-password'
```

### Grafana仪表板

模板文件：dashboard/dashboard.json

导入模板文件即可

![image-20250617144738491](https://lsky-img.hzbb.top/EAFluSPqdFTVhvgii4ENaXGjGntQVKdn/2025/06/17/68510f9666007.png)

![image-20250617144805212](https://lsky-img.hzbb.top/EAFluSPqdFTVhvgii4ENaXGjGntQVKdn/2025/06/17/68510fa0da3f2.png)



## 🔍 故障排除

### 常见问题

#### 1. 配置文件未加载
**问题**: 配置文件放在web目录但未生效
**解决**: 
- 检查文件扩展名是否为`.yaml`或`.yml`
- 确认YAML格式正确，可使用在线YAML验证器
- 查看启动日志确认配置加载情况

#### 2. 请求超时
**问题**: 监控的服务响应慢或超时
**解决**: 
- 增加`timeout_seconds`配置值
- 检查网络连接和目标服务状态
- 适当增加`interval_seconds`减少请求频率

#### 3. 认证失败
**问题**: 需要认证的API返回401/403
**解决**: 
- 检查`headers`中的认证信息是否正确
- 确认API密钥或Token是否有效
- 验证请求头格式是否符合API要求

#### 4. 内存使用过高
**问题**: 长时间运行后内存占用增加
**解决**: 
- 减少监控站点数量或增加监控间隔
- 检查是否有配置错误导致的无限重试
- 考虑重启服务释放内存

### 日志调试

启用详细日志：
```bash
# 设置日志级别
export LOG_LEVEL=debug
./services-status-exporter
```

查看实时日志：
```bash
# Docker容器日志
docker logs -f services-status-exporter

# Docker Compose日志
docker-compose logs -f services-status-exporter
```



## 📄 许可证

本项目采用 Apache License 2.0 开源许可证。详细信息请查看 [LICENSE](LICENSE) 文件。

```
Copyright 2024

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```