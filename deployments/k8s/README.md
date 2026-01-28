# Kubernetes 部署指南

本文档介绍如何在 Kubernetes 集群中部署 Asynq-Hub 系统。

## 前置要求

- Kubernetes 集群 1.24+
- kubectl CLI 工具
- Helm 3.0+ (可选)
- 已部署的 Redis 和 PostgreSQL 服务

## 部署方式

Asynq-Hub 提供两种部署方式：

1. **使用 Kustomize**（推荐用于灵活配置）
2. **使用 Helm Chart**（推荐用于快速部署）

---

## 方式一：使用 Kustomize 部署

### 1. 构建 Docker 镜像

```bash
# 构建后端镜像
cd backend
docker build -t asynqhub-backend:latest .

# 构建 Worker 镜像
cd ../worker-simulator
docker build -t asynqhub-worker:latest .
```

### 2. 推送镜像到镜像仓库

```bash
# 标记并推送到你的镜像仓库
docker tag asynqhub-backend:latest your-registry.com/asynqhub-backend:latest
docker push your-registry.com/asynqhub-backend:latest

docker tag asynqhub-worker:latest your-registry.com/asynqhub-worker:latest
docker push your-registry.com/asynqhub-worker:latest
```

### 3. 更新镜像地址

编辑 `k8s/base/backend-deployment.yaml` 和 `k8s/base/worker-deployment.yaml`，将 `image` 字段更新为你的镜像地址。

### 4. 配置 Secret

编辑 `k8s/base/backend-secret.yaml`，更新 Redis 和 PostgreSQL 连接信息：

```yaml
stringData:
  REDIS_ADDR: "redis://your-redis-host:6379/0"
  POSTGRES_DSN: "postgresql://user:password@your-pg-host:5432/asynqhub?sslmode=disable"
```

### 5. 部署到开发环境

```bash
# 创建 namespace
kubectl create namespace asynqhub-dev

# 部署
kubectl apply -k k8s/overlays/dev

# 查看部署状态
kubectl get pods -n asynqhub-dev
```

### 6. 部署到生产环境

```bash
# 创建 namespace
kubectl create namespace asynqhub-prod

# 部署
kubectl apply -k k8s/overlays/prod

# 查看部署状态
kubectl get pods -n asynqhub-prod
```

### 7. 访问服务

```bash
# 端口转发到本地访问
kubectl port-forward -n asynqhub-dev svc/asynqhub-backend 28080:28080
```

---

## 方式二：使用 Helm Chart 部署

### 1. 自定义配置

创建 `values-custom.yaml` 文件：

```yaml
backend:
  image:
    repository: your-registry.com/asynqhub-backend
    tag: "latest"
  
  replicaCount: 3
  
  redis:
    host: your-redis-host
    port: 6379
    database: 0
  
  postgres:
    host: your-pg-host
    port: 5432
    database: asynqhub
    username: postgres

worker:
  image:
    repository: your-registry.com/asynqhub-worker
    tag: "latest"
  
  replicaCount: 2
  
  config:
    workerName: "worker-1"
  
  redis:
    host: your-redis-host
    port: 6379
    database: 0
```

### 2. 安装 Chart

```bash
# 安装到开发环境
helm install asynqhub ./helm/asynq-hub \
  --namespace asynqhub-dev \
  --create-namespace \
  --values values-custom.yaml

# 安装到生产环境
helm install asynqhub ./helm/asynq-hub \
  --namespace asynqhub-prod \
  --create-namespace \
  --values values-custom.yaml \
  --set backend.replicaCount=5 \
  --set worker.replicaCount=10
```

### 3. 查看部署状态

```bash
# 查看 release 状态
helm status asynqhub -n asynqhub-dev

# 查看 pods
kubectl get pods -n asynqhub-dev
```

### 4. 更新部署

```bash
# 更新配置
helm upgrade asynqhub ./helm/asynq-hub \
  --namespace asynqhub-dev \
  --values values-custom.yaml
```

### 5. 卸载

```bash
helm uninstall asynqhub -n asynqhub-dev
```

---

## 健康检查验证

### 验证后端健康状态

```bash
# 存活检查
kubectl exec -it -n asynqhub-dev deployment/asynqhub-backend -- wget -qO- http://localhost:28080/healthz

# 就绪检查
kubectl exec -it -n asynqhub-dev deployment/asynqhub-backend -- wget -qO- http://localhost:28080/readyz
```

---

## HPA 自动扩缩容

系统默认启用 HPA，可以根据 CPU 和内存使用率自动扩缩容：

```bash
# 查看 HPA 状态
kubectl get hpa -n asynqhub-dev

# 查看详细信息
kubectl describe hpa asynqhub-backend -n asynqhub-dev
```

---

## 故障排查

### 查看日志

```bash
# 后端日志
kubectl logs -f -n asynqhub-dev deployment/asynqhub-backend

# Worker 日志
kubectl logs -f -n asynqhub-dev deployment/asynqhub-worker
```

### 查看事件

```bash
kubectl get events -n asynqhub-dev --sort-by='.lastTimestamp'
```

### 进入容器调试

```bash
kubectl exec -it -n asynqhub-dev deployment/asynqhub-backend -- sh
```

---

## 资源清理

### 清理 Kustomize 部署

```bash
kubectl delete -k k8s/overlays/dev
kubectl delete namespace asynqhub-dev
```

### 清理 Helm 部署

```bash
helm uninstall asynqhub -n asynqhub-dev
kubectl delete namespace asynqhub-dev
```

---

## 生产环境建议

1. **使用外部数据库**：建议使用云托管的 Redis 和 PostgreSQL 服务
2. **配置资源限制**：根据实际负载调整 CPU 和内存限制
3. **启用持久化存储**：为 Redis 和 PostgreSQL 配置持久化存储
4. **配置 Ingress**：使用 Ingress 暴露服务
5. **监控告警**：集成 Prometheus 和 Grafana
6. **日志收集**：集成 ELK 或 Loki
7. **备份策略**：定期备份 PostgreSQL 数据库
8. **安全加固**：
   - 使用 NetworkPolicy 限制网络访问
   - 启用 Pod Security Policy
   - 定期更新镜像和依赖
9. **多可用区部署**：配置 Pod Anti-Affinity 确保高可用
10. **滚动更新策略**：配置合理的 RollingUpdate 参数

---

## 参考资料

- [Kustomize 官方文档](https://kustomize.io/)
- [Helm 官方文档](https://helm.sh/docs/)
- [Kubernetes 最佳实践](https://kubernetes.io/docs/concepts/configuration/overview/)
