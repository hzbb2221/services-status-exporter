/*
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
*/

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

const version = "1.0.0"

// WebConfig Web服务器配置结构
type WebConfig struct {
	TLSServerConfig struct {
		CertFile string `yaml:"cert_file"`
		KeyFile  string `yaml:"key_file"`
	} `yaml:"tls_server_config"`
	BasicAuthUsers map[string]string `yaml:"basic_auth_users"`
}

// SiteConfig 站点配置结构
type SiteConfig struct {
	URL             string            `yaml:"url"`
	Headers         map[string]string `yaml:"headers"`
	Body            string            `yaml:"body"`
	IntervalSeconds int               `yaml:"interval_seconds"`
	TimeoutSeconds  int               `yaml:"timeout_seconds"`
	SkipVerify      bool              `yaml:"skip_verify"` // 是否跳过HTTPS证书验证
	Labels          map[string]string `yaml:"labels"`
}

var (
	// 监控指标
	requestDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "site_request_duration_seconds",
			Help: "HTTP request duration in seconds",
		},
		[]string{"site", "method", "env", "service", "type"},
	)

	statusCode = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "site_status_code",
			Help: "HTTP response status code",
		},
		[]string{"site", "method", "env", "service", "type"},
	)

	requestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "site_request_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"site", "method", "status", "env", "service", "type"},
	)

	siteUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "site_up",
			Help: "Whether the site is up (1) or down (0)",
		},
		[]string{"site", "method", "env", "service", "type"},
	)

	// 应用程序运行状态指标
	appInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "services_status_exporter_info",
			Help: "Information about the services status exporter application",
		},
		[]string{"version", "build_time"},
	)

	appUp = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "services_status_exporter_up",
			Help: "Whether the services status exporter is running (1) or not (0)",
		},
	)

	webConfigMutex sync.Mutex
	webConfig      *WebConfig
	siteConfigs    []SiteConfig
)

func init() {
	// 注册监控指标
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(statusCode)
	prometheus.MustRegister(requestTotal)
	prometheus.MustRegister(siteUp)
	prometheus.MustRegister(appInfo)
	prometheus.MustRegister(appUp)
}

// 加载站点配置
func loadSiteConfigs(webDir string) ([]SiteConfig, error) {
	var configs []SiteConfig

	err := filepath.Walk(webDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".yaml" {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("读取配置文件失败 %s: %v", path, err)
			}

			var config SiteConfig
			if err := yaml.Unmarshal(data, &config); err != nil {
				return fmt.Errorf("解析配置文件失败 %s: %v", path, err)
			}

			configs = append(configs, config)
		}

		return nil
	})

	return configs, err
}

// 检查站点状态
func checkSite(config SiteConfig) {
	// 设置默认值
	if config.IntervalSeconds <= 0 {
		config.IntervalSeconds = 60 // 默认60秒
	}
	if config.TimeoutSeconds <= 0 {
		config.TimeoutSeconds = 30 // 默认30秒
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.SkipVerify,
		},
	}
	client := &http.Client{
		Timeout:   time.Duration(config.TimeoutSeconds) * time.Second,
		Transport: tr,
	}

	// 初始化method为GET，后续可能会根据配置修改
	method := "GET"

	labels := []string{config.URL, method}
	for _, labelKey := range []string{"env", "service", "type"} {
		if labelValue, exists := config.Labels[labelKey]; exists {
			labels = append(labels, labelValue)
		} else {
			labels = append(labels, "")
		}
	}

	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("站点检查发生严重错误 %s: %v\n", config.URL, r)
					siteUp.WithLabelValues(labels...).Set(0)
				}
				time.Sleep(time.Duration(config.IntervalSeconds) * time.Second)
			}()

			req, err := http.NewRequest("GET", config.URL, nil)
			if err != nil {
				log.Printf("创建请求失败 %s: %v\n", config.URL, err)
				siteUp.WithLabelValues(labels...).Set(0)
				requestTotal.WithLabelValues(append([]string{config.URL, method, "error"}, labels[2:]...)...).Inc()
				return
			}

			// 添加请求头
			for key, value := range config.Headers {
				req.Header.Add(key, value)
			}

			// 如果有请求体，设置POST方法
			if config.Body != "" {
				req.Method = "POST"
				method = "POST"
				// 更新labels中的method
				labels[1] = method
				req.Body = ioutil.NopCloser(strings.NewReader(config.Body))
			}

			start := time.Now()
			resp, err := client.Do(req)

			if err != nil {
				log.Printf("请求失败 %s: %v\n", config.URL, err)
				siteUp.WithLabelValues(labels...).Set(0)
				requestTotal.WithLabelValues(append([]string{config.URL, method, "error"}, labels[2:]...)...).Inc()
				return
			}
			defer resp.Body.Close()

			duration := time.Since(start).Seconds()

			// 更新监控指标
			requestDuration.WithLabelValues(labels...).Set(duration)
			statusCode.WithLabelValues(labels...).Set(float64(resp.StatusCode))
			siteUp.WithLabelValues(labels...).Set(1)

			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				requestTotal.WithLabelValues(append([]string{config.URL, method, "success"}, labels[2:]...)...).Inc()
			} else {
				requestTotal.WithLabelValues(append([]string{config.URL, method, "error"}, labels[2:]...)...).Inc()
			}
		}()
	}
}

// 加载Web配置
func loadWebConfig(configFile string) (*WebConfig, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("读取Web配置文件失败: %v", err)
	}

	var config WebConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析Web配置文件失败: %v", err)
	}

	return &config, nil
}

// 基本认证中间件
func basicAuthMiddleware(handler http.Handler, users map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		hashedPassword, exists := users[user]
		if !exists {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(pass)); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func main() {
	webDir := flag.String("web-dir", "web", "Web配置目录路径")
	addr := flag.String("addr", ":9112", "HTTP服务监听地址")
	webConfigFile := flag.String("web.config.file", "web-config.yml", "Web服务器配置文件路径")
	versionFlag := flag.Bool("version", false, "显示版本信息")
	helpFlag := flag.Bool("help", false, "显示帮助信息")
	flag.Parse()

	// 设置应用程序运行状态指标
	buildTime := time.Now().Format("2006-01-02T15:04:05Z07:00")
	appInfo.WithLabelValues(version, buildTime).Set(1)
	appUp.Set(1)

	// 显示帮助信息
	if *helpFlag {
		fmt.Println("Usage: services-status-exporter [options]")
		fmt.Println("Options:")
		fmt.Println("  -web-dir <path>         Web配置目录路径 (默认: web)")
		fmt.Println("  -addr <address>         HTTP服务监听地址 (默认: :9112)")
		fmt.Println("  -web.config.file <path> Web服务器配置文件路径 (默认: web-config.yml)")
		fmt.Println("  -version                显示版本信息")
		fmt.Println("  -help                   显示帮助信息")
		return
	}

	// 显示版本信息
	if *versionFlag {
		fmt.Println("Version:", version)
		return
	}

	// 初始化配置
	var err error
	webConfig, err = loadWebConfig(*webConfigFile)
	if err != nil {
		log.Printf("加载Web配置失败: %v，将使用默认配置\n", err)
		webConfig = &WebConfig{}
	}

	siteConfigs, err = loadSiteConfigs(*webDir)
	if err != nil {
		log.Fatalf("加载配置失败: %v\n", err)
	}

	// 为每个站点启动检查协程
	for _, config := range siteConfigs {
		go checkSite(config)
	}

	// 创建metrics处理器
	metricsHandler := promhttp.Handler()

	// 如果配置了基本认证，添加认证中间件
	if len(webConfig.BasicAuthUsers) > 0 {
		metricsHandler = basicAuthMiddleware(metricsHandler, webConfig.BasicAuthUsers)
	}

	// 注册处理器
	http.Handle("/metrics", metricsHandler)

	// 启动HTTP服务器
	log.Printf("Starting server on %s\n", *addr)
	server := &http.Server{
		Addr:    *addr,
		Handler: nil,
	}

	// 如果配置了TLS，启动HTTPS服务器
	if webConfig.TLSServerConfig.CertFile != "" && webConfig.TLSServerConfig.KeyFile != "" {
		log.Printf("启用TLS加密传输\n")
		log.Fatal(server.ListenAndServeTLS(
			webConfig.TLSServerConfig.CertFile,
			webConfig.TLSServerConfig.KeyFile,
		))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}
