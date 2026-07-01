# 易扣 AI-Go 教学版 — 项目概览

> 本文档面向需要快速了解项目全貌的开发者，涵盖架构、目录、核心流程与运行方式。详细分步教程见 [teach_catalog](../teach_catalog/) 各章节。

## 1. 项目简介

**易扣 AI-Go 教学版**（模块名 `yikou-ai-go-teach`）是 [易扣AI-Go 重构版](https://github.com/FeiWuSama/yikou-ai-go) 的配套教学仓库，目标是手把手带你从零构建一个企业级 Go Web + AI 应用生成平台。

核心能力：

- 用户注册、登录、权限管理（普通用户 / 管理员）
- AI 应用（App）的创建、编辑、部署与浏览
- 基于 **CloudWeGo Eino** 的对话式代码生成（HTML / 多文件）
- 流式 SSE 响应 + Redis 对话记忆 + MySQL 对话历史持久化

## 2. 技术栈

| 类别 | 技术 |
|------|------|
| 语言 | Go 1.24+ |
| Web 框架 | [CloudWeGo Hertz](https://github.com/cloudwego/hertz) |
| ORM | GORM + GORM Gen（代码生成） |
| 数据库 | MySQL 8.x |
| 缓存 / Session | Redis |
| 依赖注入 | Google Wire |
| 配置管理 | Viper |
| AI 框架 | [CloudWeGo Eino](https://github.com/cloudwego/eino) + OpenAI 兼容 API |
| API 文档 | Swaggo |
| 前端 | Vue 3 + Vite + TypeScript + Ant Design Vue（`yikou-ai-feiwu-front/`） |

## 3. 项目目录结构

```
yikou-ai-go/
├── main.go                    # 程序入口，Wire 初始化并启动 Hertz
├── config/                    # 配置文件与 Config 结构体
│   ├── config.yml
│   └── config.go
├── wire/                      # Google Wire 依赖注入
│   ├── wire.go
│   └── wire_gen.go
├── internal/                  # 业务核心（不对外暴露）
│   ├── handler/               # HTTP 处理器（Controller 层）
│   ├── logic/                 # 业务逻辑实现（Service 实现层）
│   ├── service/               # Service 接口定义
│   ├── api/                   # 请求 / 响应 DTO
│   ├── dal/                   # 数据访问层（GORM Gen 生成）
│   │   ├── model/             # 数据库实体
│   │   ├── query/             # 类型安全查询
│   │   └── vo/                # 视图对象
│   ├── router/                # 路由注册
│   ├── middleware/            # 鉴权等中间件
│   ├── ai/                    # AI 相关
│   │   ├── agent/             # Eino Agent（代码生成）
│   │   ├── llm/               # 大模型封装
│   │   └── myprompt/          # Prompt 模板
│   ├── core/                  # 代码生成门面、解析器、保存器
│   └── store/                 # Redis 对话记忆存储
├── pkg/                       # 可复用公共包
│   ├── enum/                  # 枚举（角色、代码生成类型等）
│   ├── response/              # 统一响应结构
│   ├── errorutil/             # 业务错误
│   ├── constants/             # 常量
│   └── snowflake/             # ID 生成
├── prompt/                    # AI System Prompt 文本文件
├── sql/                       # 建表 SQL
├── docs/                      # Swagger 生成文件
├── teach_catalog/             # 分章教学文档
├── cmd/gen/                   # GORM Gen 代码生成入口
└── yikou-ai-feiwu-front/      # Vue 3 前端项目
```

## 4. 分层架构

项目采用经典分层 + 依赖注入，请求自上而下流转：

```
HTTP Request
    ↓
Router（路由 + 中间件）
    ↓
Handler（参数校验、响应封装）
    ↓
Service Interface ← Logic（业务逻辑实现）
    ↓
DAL（GORM Gen Query）→ MySQL
    ↓
AI Agent / Redis / 文件系统（代码生成场景）
```

**Wire 依赖图**（`wire/wire.go`）将 Config、DB、Redis、LLM、Service、Handler 等组件自动装配，入口 `main.go` 仅调用 `wire.InitializeApp()` 启动服务。

## 5. 核心模块说明

### 5.1 用户模块（User）

- 注册、登录、登出
- Session 存 Redis，Cookie 携带 `sessionId`
- 角色：`user`（普通用户）、`admin`（管理员）
- 管理员可 CRUD 用户、分页查询

### 5.2 应用模块（App）

- 用户创建 AI 应用，配置名称、封面、初始 Prompt、代码生成类型
- 支持「精选应用」公开展示
- 核心接口：`GET /app/chat/gen/code` — SSE 流式对话生成代码
- 生成类型（`pkg/enum/code_gentype.go`）：
  - `html` — 单文件 HTML
  - `multi_file` — HTML + CSS + JS 多文件
  - `vue_project` — Vue 工厂模式（枚举已预留）

### 5.3 对话历史模块（Chat History）

- 每次 AI 对话写入 `chat_history` 表
- 启动 Agent 时从 DB 加载最近 N 条记录到 Redis 记忆（`CodeGenAgentFactory`）
- Eino Agent 流式生成时同步维护 Redis 短期记忆（`store.RedisMemoryStore`）

### 5.4 AI 代码生成流程

```
用户消息 (SSE)
    ↓
AppHandler.ChatToGenCode
    ↓
AppService → YiKouAiCodegenFacade.GenCodeStreamAndSave
    ↓
CodeGenAgentFactory.GetCodeGenAgent(appId, type)
    ├── 从 DB 加载历史 → Redis MemoryStore
    └── 缓存 Agent 实例（go-cache，最多 1000 个）
    ↓
CodeGenAgent.GenerateXxxStream（Eino ADK + ChatModel）
    ↓
流复制：一路 SSE 返回前端，一路异步解析 + 保存文件
    ├── parser.CodeParserExecutor
    └── saver.CodeFileSaverExecutor
```

关键类：

- `internal/core/ai_codegen_facade.go` — 代码生成统一门面
- `internal/ai/agent/codegen_agent.go` — 基于 Eino ADK 的生成 Agent
- `internal/ai/agent/base_agent.go` — 同步 / 流式生成、Redis 记忆读写

## 6. 数据模型

| 表名 | 说明 | 主要字段 |
|------|------|----------|
| `user` | 用户 | userAccount, userPassword, userRole |
| `app` | AI 应用 | appName, initPrompt, codeGenType, deployKey, userId |
| `chat_history` | 对话历史 | message, messageType(user/ai), appId, turnNumber |

建表脚本：`sql/create_table.sql`

## 7. API 路由概览

基础路径：`/api`（可在 `config/config.yml` 配置）

| 模块 | 路径前缀 | 说明 |
|------|----------|------|
| 健康检查 | `GET /ping` | 无需鉴权 |
| Swagger | `GET /swagger/*` | API 文档 |
| 用户 | `/user/*` | 注册、登录、用户管理 |
| 应用 | `/app/*` | 应用 CRUD、流式代码生成 |
| 对话历史 | `/chatHistory/*` | 按应用查历史、管理员分页 |

鉴权：`middleware.AuthMiddleware` 校验 Cookie Session + 角色等级。

## 8. 配置与启动

### 8.1 配置文件

编辑 `config/config.yml`：

```yaml
server:
  port: 8123
  context_path: /api

database:
  host: localhost
  port: 3306
  username: root
  password: your_password
  database: yikou_ai

ai:
  chat-model:
    base-url: https://dashscope.aliyuncs.com/compatible-mode/v1
    api-key: <你的api-key>
    model-name: qwen3.5-plus

redis:
  host: localhost
  port: 6379
```

支持多环境：启动时加 `-env=local` 会读取 `config/config-local.yml`。

### 8.2 后端启动

```bash
# 1. 执行建表 SQL
mysql -u root -p < sql/create_table.sql

# 2. 安装依赖
go mod download

# 3. 生成 Wire 代码（修改 wire.go 后需要）
go generate ./wire/...

# 4. 启动服务
go run main.go
```

服务默认监听 `http://localhost:8123`，Swagger 地址：`http://localhost:8123/api/swagger/index.html`

### 8.3 前端启动

```bash
cd yikou-ai-feiwu-front
npm install
npm run dev
```

前端通过 OpenAPI 生成 API 客户端（`openapi2ts.config.ts`），对接后端 `/api` 接口。

## 9. 前端页面结构

| 页面 | 路径 | 功能 |
|------|------|------|
| 首页 | `/` | 精选应用展示 |
| 用户登录/注册 | `/user/login`, `/user/register` | 账号体系 |
| 应用对话 | `/app/chat/:id` | AI 对话 + 代码预览 |
| 应用编辑 | `/app/edit/:id` | 应用配置 |
| 用户中心 | `/user/center` | 我的应用 |
| 管理后台 | `/admin/*` | 用户、应用、对话管理 |

## 10. 教学章节索引

| 章节 | 主题 |
|------|------|
| [第1章](../teach_catalog/chapter_1.md) | 环境准备与依赖整合 |
| [第2章](../teach_catalog/chapter_2.md) | 用户模块搭建 |
| [第3章](../teach_catalog/chapter_3.md) | Eino AI 应用生成逻辑 |
| [第4章](../teach_catalog/chapter_4.md) | 应用模块搭建 |
| [第5章](../teach_catalog/chapter_5.md) | 对话历史与 Eino 记忆 |

## 11. 常用开发命令

```bash
# GORM Gen 重新生成 DAL 代码
go run cmd/gen/main.go

# 生成 / 更新 Swagger 文档
swag init

# 运行测试
go test ./...
```

## 12. 设计要点小结

1. **Wire 依赖注入**：避免手动组装，组件边界清晰，便于测试与扩展。
2. **Interface + Logic**：Handler 依赖 Service 接口，Logic 包实现具体业务，符合依赖倒置。
3. **GORM Gen**：类型安全查询，减少手写 SQL 错误。
4. **Eino ADK**：Agent 编排、流式输出、Prompt 模板统一管理。
5. **双层记忆**：Redis 存 Agent 短期上下文，MySQL 存完整对话历史，启动时回填。
6. **流式双路处理**：SSE 实时返回给用户，后台 goroutine 解析并落盘代码文件。

---

## 13. Kubernetes 部署

完整 K8s 部署、访问、重建与故障排查说明见 **[K8S_DEPLOY.md](./K8S_DEPLOY.md)**。

快速启动：

```powershell
minikube start --driver=docker --image-mirror-country=cn --preload=false
powershell -ExecutionPolicy Bypass -File .\k8s\deploy.ps1
kubectl port-forward -n yikou-ai svc/frontend 30080:80
kubectl port-forward -n yikou-ai svc/backend 30123:8123
```

- 前端：http://localhost:30080
- 后端：http://localhost:30123/api/ping

---

如有问题，可参考各章 `teach_catalog` 详细教程，或查阅 Swagger 在线文档。
