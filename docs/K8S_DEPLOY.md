# Kubernetes 部署与运维指南

本文档说明如何在本地 Docker 环境中使用 Minikube 部署、访问和重建 **易扣 AI-Go** 项目。

## 1. 前置条件

| 工具 | 说明 |
|------|------|
| Docker Desktop | 已启动 |
| kubectl | 已安装 |
| minikube | 已安装（推荐 v1.38+） |

验证：

```powershell
docker info
kubectl version --client
minikube version
```

## 2. 目录结构

```
k8s/
├── base/                          # 基础资源（Deployment、Service、Secret 等）
├── overlays/
│   ├── local/                     # 本地开发（NodePort 暴露端口）
│   └── prod/                      # 生产环境
├── deploy.ps1                     # Windows 一键部署脚本
└── deploy.sh                      # Linux/macOS 一键部署脚本
```

本地 overlay 通过 NodePort 暴露以下端口：

| 服务 | NodePort | 说明 |
|------|----------|------|
| frontend | 30080 | 前端 Nginx |
| backend | 30123 | 后端 API |
| mysql | 30306 | MySQL 8.x |
| redis | 30379 | Redis 7 |

## 3. 首次启动集群

### 3.1 启动 Minikube

```powershell
# 国内网络建议加镜像加速
minikube start --driver=docker --cpus=4 --memory=6144 `
  --image-mirror-country=cn --preload=false
```

若拉取 `kicbase` 镜像失败，可先手动拉取并打标签：

```powershell
docker pull registry.cn-hangzhou.aliyuncs.com/google_containers/kicbase:v0.0.50
docker tag registry.cn-hangzhou.aliyuncs.com/google_containers/kicbase:v0.0.50 `
  gcr.io/k8s-minikube/kicbase:v0.0.50

minikube start --driver=docker --cpus=4 --memory=6144 `
  --image-mirror-country=cn `
  --base-image=gcr.io/k8s-minikube/kicbase:v0.0.50 `
  --preload=false
```

### 3.2 一键部署项目

在项目根目录执行：

```powershell
powershell -ExecutionPolicy Bypass -File .\k8s\deploy.ps1
```

Linux / macOS：

```bash
chmod +x k8s/deploy.sh
./k8s/deploy.sh
```

脚本会自动完成：

1. 构建 `yikou-ai-go-backend:latest`、`yikou-ai-go-frontend:latest`
2. 将镜像导入 minikube
3. 应用 `k8s/overlays/local` 配置
4. 等待 mysql、redis、backend、frontend 就绪

## 4. 重新构建与启动

### 4.1 完整重建（改代码后推荐）

```powershell
minikube start --driver=docker --image-mirror-country=cn --preload=false
powershell -ExecutionPolicy Bypass -File .\k8s\deploy.ps1
```

### 4.2 只重启 Pod（不重新构建镜像）

```powershell
kubectl rollout restart deployment -n yikou-ai
```

或单独重启某个服务：

```powershell
kubectl rollout restart deployment/backend -n yikou-ai
kubectl rollout restart deployment/frontend -n yikou-ai
```

### 4.3 只更新 K8s 配置（不改代码）

```powershell
kubectl apply -k .\k8s\overlays\local
```

### 4.4 强制使用新镜像

后端/前端 Deployment 使用 `imagePullPolicy: IfNotPresent`，更新镜像后需重启 Pod：

```powershell
powershell -ExecutionPolicy Bypass -File .\k8s\deploy.ps1
kubectl rollout restart deployment/backend deployment/frontend -n yikou-ai
```

## 5. 访问服务

### 5.1 Windows + Minikube Docker 驱动

NodePort **不能**直接用 `localhost` 访问，需使用以下方式之一。

**方式 A：端口转发（推荐，端口固定）**

```powershell
kubectl port-forward -n yikou-ai svc/frontend 30080:80
kubectl port-forward -n yikou-ai svc/backend 30123:8123
kubectl port-forward -n yikou-ai svc/mysql 30306:3306
kubectl port-forward -n yikou-ai svc/redis 30379:6379
```

**方式 B：minikube service（自动分配临时端口）**

```powershell
minikube service frontend -n yikou-ai
minikube service backend -n yikou-ai --url
```

### 5.2 访问地址

| 服务 | 地址 | 备注 |
|------|------|------|
| 前端 | http://localhost:30080 | 需 port-forward |
| 后端 API | http://localhost:30123/api | 需 port-forward |
| 健康检查 | http://localhost:30123/api/ping | |
| Swagger | http://localhost:30123/api/swagger/index.html | |
| MySQL | 127.0.0.1:30306 | 需 port-forward |
| Redis | 127.0.0.1:30379 | 需 port-forward |

### 5.3 外部客户端连接 MySQL / Redis

**MySQL（DBeaver / Navicat 等）**

| 字段 | 值 |
|------|-----|
| Host | 127.0.0.1 |
| Port | 30306 |
| User | root |
| Password | yikou123456 |
| Database | yikou_ai |

**Redis（Redis Insight 等）**

| 字段 | 值 |
|------|-----|
| Host | 127.0.0.1 |
| Port | 30379 |
| Password | （无） |

> 连接前请先执行对应的 `kubectl port-forward` 命令。

## 6. 配置 AI API Key

K8s 中默认 Key 为占位值 `changeme`，需手动更新：

```powershell
kubectl edit secret backend-secret -n yikou-ai
```

修改 `AI_API_KEY` 后重启 backend：

```powershell
kubectl rollout restart deployment/backend -n yikou-ai
```

## 7. 常用命令速查

以下命令默认命名空间为 `yikou-ai`，可在项目根目录直接执行。

### 7.1 集群管理（Minikube）

```powershell
# 查看集群状态
minikube status

# 启动集群
minikube start --driver=docker --image-mirror-country=cn --preload=false

# 停止集群（保留数据，下次 start 可恢复）
minikube stop

# 暂停集群（类似休眠）
minikube pause

# 恢复暂停的集群
minikube unpause

# 修复 kubectl 上下文（集群端口变化时使用）
minikube update-context

# 查看集群 IP
minikube ip

# 打开 K8s 控制台
minikube dashboard

# 删除整个集群（所有数据清空，不可恢复）
minikube delete

# 删除并重建集群
minikube delete
minikube start --driver=docker --cpus=4 --memory=6144 `
  --image-mirror-country=cn --preload=false
```

### 7.2 部署与重启

```powershell
# 一键构建镜像 + 部署（改代码后推荐）
powershell -ExecutionPolicy Bypass -File .\k8s\deploy.ps1

# 只应用 K8s 配置（不改镜像）
kubectl apply -k .\k8s\overlays\local

# 重启全部服务
kubectl rollout restart deployment -n yikou-ai

# 重启单个服务
kubectl rollout restart deployment/backend -n yikou-ai
kubectl rollout restart deployment/frontend -n yikou-ai
kubectl rollout restart deployment/mysql -n yikou-ai
kubectl rollout restart deployment/redis -n yikou-ai

# 重启 backend + frontend（更新镜像后）
kubectl rollout restart deployment/backend deployment/frontend -n yikou-ai

# 查看滚动更新状态
kubectl rollout status deployment/backend -n yikou-ai

# 回滚到上一版本
kubectl rollout undo deployment/backend -n yikou-ai
kubectl rollout undo deployment/frontend -n yikou-ai

# 查看历史版本
kubectl rollout history deployment/backend -n yikou-ai
```

### 7.3 删除资源

```powershell
# 删除整个项目命名空间（所有 Pod、Service、PVC 一并删除）
kubectl delete namespace yikou-ai

# 删除后重新部署
kubectl apply -k .\k8s\overlays\local

# 删除单个 Deployment（Pod 会重建，若 replicas>0）
kubectl delete deployment backend -n yikou-ai
kubectl delete deployment frontend -n yikou-ai
kubectl delete deployment mysql -n yikou-ai
kubectl delete deployment redis -n yikou-ai

# 删除 Service
kubectl delete svc backend -n yikou-ai

# 删除 Secret / ConfigMap
kubectl delete secret backend-secret -n yikou-ai
kubectl delete secret mysql-secret -n yikou-ai
kubectl delete configmap mysql-init-sql -n yikou-ai

# 删除 PVC（会清空持久化数据，MySQL 数据也会丢失）
kubectl delete pvc mysql-pvc -n yikou-ai
kubectl delete pvc backend-code-output-pvc -n yikou-ai

# 删除单个 Pod（Deployment 会自动重建）
kubectl delete pod -n yikou-ai -l app=backend
kubectl get pods -n yikou-ai -o name | ForEach-Object { kubectl delete $_ -n yikou-ai }

# 强制删除卡住的 Pod
kubectl delete pod <pod-name> -n yikou-ai --force --grace-period=0
```

### 7.4 查看状态

```powershell
# 查看所有 Pod
kubectl get pods -n yikou-ai

# 查看 Pod 详情（排查 Pending / CrashLoopBackOff）
kubectl get pods -n yikou-ai -o wide
kubectl describe pod -n yikou-ai -l app=backend

# 查看 Service
kubectl get svc -n yikou-ai

# 查看 Deployment
kubectl get deployment -n yikou-ai

# 查看 PVC（持久化存储）
kubectl get pvc -n yikou-ai

# 查看 Secret / ConfigMap
kubectl get secret -n yikou-ai
kubectl get configmap -n yikou-ai

# 查看 Ingress
kubectl get ingress -n yikou-ai

# 查看所有资源
kubectl get all -n yikou-ai

# 查看事件（排查启动失败）
kubectl get events -n yikou-ai --sort-by='.lastTimestamp'
```

### 7.5 日志与调试

```powershell
# 查看 backend 最近 100 行日志
kubectl logs -n yikou-ai deploy/backend --tail=100

# 实时跟踪日志
kubectl logs -n yikou-ai deploy/backend -f

# 查看 frontend / mysql / redis 日志
kubectl logs -n yikou-ai deploy/frontend --tail=100
kubectl logs -n yikou-ai deploy/mysql --tail=100
kubectl logs -n yikou-ai deploy/redis --tail=100

# 查看崩溃前的日志
kubectl logs -n yikou-ai -l app=backend --previous

# 进入容器 Shell
kubectl exec -it -n yikou-ai deploy/backend -- sh
kubectl exec -it -n yikou-ai deploy/mysql -- bash
kubectl exec -it -n yikou-ai deploy/redis -- sh

# 在 backend 容器内测试 API
kubectl exec -n yikou-ai deploy/backend -- wget -qO- http://127.0.0.1:8123/api/ping

# 在集群内测试 MySQL 连通性
kubectl exec -n yikou-ai deploy/mysql -- mysqladmin ping -uroot -pyikou123456
```

### 7.6 端口转发

```powershell
# 前端
kubectl port-forward -n yikou-ai svc/frontend 30080:80

# 后端
kubectl port-forward -n yikou-ai svc/backend 30123:8123

# MySQL
kubectl port-forward -n yikou-ai svc/mysql 30306:3306

# Redis
kubectl port-forward -n yikou-ai svc/redis 30379:6379

# 转发到指定 Pod（绕过 Service）
kubectl port-forward -n yikou-ai pod/<pod-name> 8123:8123
```

> `port-forward` 为前台进程，关闭终端即断开。需要长期转发可另开终端窗口，或使用 `Start-Process` 后台运行。

### 7.7 配置与密钥

```powershell
# 编辑 AI API Key
kubectl edit secret backend-secret -n yikou-ai

# 编辑 MySQL 密码
kubectl edit secret mysql-secret -n yikou-ai

# 查看 Secret 内容（Base64 解码）
kubectl get secret backend-secret -n yikou-ai -o jsonpath='{.data.AI_API_KEY}' | ForEach-Object { [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($_)) }

# 修改配置后重启 backend 生效
kubectl rollout restart deployment/backend -n yikou-ai
```

### 7.8 扩缩容

```powershell
# 扩展 frontend 副本数
kubectl scale deployment frontend -n yikou-ai --replicas=2

# 缩容到 1 个副本
kubectl scale deployment frontend -n yikou-ai --replicas=1

# 停止某个服务（缩容到 0，不删除 Deployment）
kubectl scale deployment frontend -n yikou-ai --replicas=0

# 恢复服务
kubectl scale deployment frontend -n yikou-ai --replicas=1
```

### 7.9 镜像管理

```powershell
# 本地构建镜像
docker build -t yikou-ai-go-backend:latest .
docker build -f yikou-ai-feiwu-front/Dockerfile.prod -t yikou-ai-go-frontend:latest yikou-ai-feiwu-front

# 导入镜像到 minikube
minikube image load yikou-ai-go-backend:latest
minikube image load yikou-ai-go-frontend:latest

# 查看 minikube 内的镜像
minikube ssh -- docker images | findstr yikou

# 删除 minikube 内的旧镜像
minikube ssh -- docker rmi yikou-ai-go-backend:latest
```

### 7.10 典型场景速查

| 场景 | 命令 |
|------|------|
| 改代码后重新部署 | `powershell -ExecutionPolicy Bypass -File .\k8s\deploy.ps1` |
| 只重启 backend | `kubectl rollout restart deployment/backend -n yikou-ai` |
| 清空数据库重来 | `kubectl delete pvc mysql-pvc -n yikou-ai` 后重新 `kubectl apply -k .\k8s\overlays\local` |
| 完全卸载项目 | `kubectl delete namespace yikou-ai` |
| 完全卸载集群 | `minikube delete` |
| 集群起不来 | `minikube update-context` → `minikube start ...` |
| 查看 backend 报错 | `kubectl logs -n yikou-ai deploy/backend --tail=100` |
| 本地访问前端 | `kubectl port-forward -n yikou-ai svc/frontend 30080:80` |

## 8. 故障排查

### kubeconfig 过期 / 集群不可达

```powershell
minikube update-context
minikube start --driver=docker --image-mirror-country=cn --preload=false
```

### Pod 一直 Pending / CrashLoopBackOff

```powershell
kubectl describe pod -n yikou-ai -l app=backend
kubectl logs -n yikou-ai -l app=backend --previous
```

### 镜像更新后 Pod 仍用旧代码

确认已执行 `deploy.ps1` 导入新镜像，并重启 Deployment：

```powershell
kubectl rollout restart deployment/backend deployment/frontend -n yikou-ai
```

### 完全重建集群

```powershell
minikube delete
minikube start --driver=docker --cpus=4 --memory=6144 `
  --image-mirror-country=cn --preload=false
powershell -ExecutionPolicy Bypass -File .\k8s\deploy.ps1
```

## 9. 与 Docker Compose 的区别

| 对比项 | Docker Compose | Kubernetes (Minikube) |
|--------|----------------|------------------------|
| 启动命令 | `docker compose up -d` | `k8s/deploy.ps1` |
| 前端端口 | 5173 | 30080（需 port-forward） |
| 后端端口 | 8123 | 30123（需 port-forward） |
| MySQL 端口 | 3307 | 30306（需 port-forward） |
| Redis 端口 | 6380 | 30379（需 port-forward） |
| 数据 | 独立 volume | 独立 PVC |

两套环境数据**不互通**，请勿同时占用相同本地端口。

## 10. 生产环境

生产部署使用 `prod` overlay：

```powershell
powershell -ExecutionPolicy Bypass -File .\k8s\deploy.ps1 prod
```

生产环境通过 Ingress 对外暴露，详见 `k8s/base/ingress.yaml`。
