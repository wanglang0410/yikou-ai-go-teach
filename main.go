package main

import (
	"flag"
	"fmt"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/swagger"
	"github.com/hertz-contrib/swagger/example/basic/docs"
	"strconv"
	"yikou-ai-go-teach/config"
	"yikou-ai-go-teach/internal/router"
)

// initServer 初始化 Web 服务器
func initServer() *server.Hertz {
	cfg := config.GlobalConfig

	// 动态设置 Swagger 信息
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%d", cfg.Server.Port)
	docs.SwaggerInfo.BasePath = cfg.Server.ContextPath

	// 初始化swagger路径
	swaggerPath := fmt.Sprintf("http://localhost:%d%s/swagger/doc.json", cfg.Server.Port, cfg.Server.ContextPath)
	url := swagger.URL(swaggerPath)

	// 创建 Hertz 服务器
	h := server.Default(
		server.WithHostPorts(":"+strconv.Itoa(cfg.Server.Port)),
		server.WithBasePath(cfg.Server.ContextPath),
	)

	// 注册路由
	router.RegisterRoutes(h, url)
	return h
}

func main() {
	// 解析命令行参数
	env := flag.String("env", "", "运行环境，如 local, dev, test, prod")
	flag.Parse()

	// 初始化配置
	// 如果不指定 -env 参数，默认读取 config.yaml
	// 如果指定 -env local，则读取 config-local.yaml
	// 配置文件路径会自动从项目根目录下的 config 目录读取
	config.InitConfig(*env)

	// 初始化 Web 服务器
	h := initServer()

	// 启动服务器
	h.Spin()
}
