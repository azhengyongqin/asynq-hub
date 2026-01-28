.PHONY: help build build-web embed-web build-all test lint clean docker-build run swagger swagger-view swagger-fmt

help: ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## 编译服务
	go build -o bin/server ./cmd/server
	go build -o bin/example ./cmd/example

build-web: ## 构建前端静态文件
	cd web && npm run build
	@echo "✅ 前端构建完成"

embed-web: build-web ## 构建前端并嵌入到 Go 服务
	rm -rf cmd/server/webui
	cp -r web/dist cmd/server/webui
	@echo "✅ 前端文件已复制到 cmd/server/webui"
	@echo "📦 重新编译以嵌入静态文件..."
	go build -o bin/server ./cmd/server
	@echo "✅ Web UI 已嵌入到 server"
	@echo "🌐 访问地址:"
	@echo "   - Web UI: http://localhost:28080/"
	@echo "   - API 文档: http://localhost:28080/swagger/index.html"
	@echo "   - API 端点: http://localhost:28080/api/v1/"

build-all: embed-web build ## 构建前端和后端（包含嵌入）
	go build -o bin/example ./cmd/example
	@echo "✅ 所有服务编译完成"

test: ## 运行测试
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## 运行测试并生成覆盖率报告
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

lint: ## 运行代码检查
	golangci-lint run ./...

fmt: ## 格式化代码
	go fmt ./...

swagger: ## 生成 Swagger API 文档
	swag init -g cmd/server/main.go -o docs
	@echo "✅ Swagger 文档已生成到 docs/ 目录"
	@echo "📖 查看文档: http://localhost:28080/swagger/index.html"

swagger-view: swagger ## 生成 Swagger 文档并在浏览器中打开
	@echo "正在打开 Swagger 文档..."
	@sleep 1
	@if command -v open > /dev/null; then \
		open http://localhost:28080/swagger/index.html; \
	elif command -v xdg-open > /dev/null; then \
		xdg-open http://localhost:28080/swagger/index.html; \
	else \
		echo "请手动访问: http://localhost:28080/swagger/index.html"; \
	fi

swagger-fmt: ## 格式化 Swagger 注释
	swag fmt -g cmd/server/main.go

clean: ## 清理编译产物
	rm -rf bin/
	rm -f coverage.out coverage.html

docker-build: ## 构建 Docker 镜像
	docker build -t asynqhub-server:latest -f deployments/docker/Dockerfile.server .
	docker build -t asynqhub-example:latest -f deployments/docker/Dockerfile.example .

docker-compose-up: ## 启动本地开发环境
	docker-compose up -d

docker-compose-down: ## 停止本地开发环境
	docker-compose down

run: ## 运行服务
	go run ./cmd/server

run-example: ## 运行示例 Worker
	go run ./cmd/example

migrate: ## 应用数据库迁移
	cd prisma && npx prisma migrate deploy

install-tools: ## 安装开发工具
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

security: ## 运行安全检查
	govulncheck ./...
	trivy fs --severity HIGH,CRITICAL .

k8s-deploy-dev: ## 部署到 K8s 开发环境
	kubectl apply -k deployments/k8s/overlays/dev

k8s-deploy-prod: ## 部署到 K8s 生产环境
	kubectl apply -k deployments/k8s/overlays/prod

helm-install-dev: ## 使用 Helm 安装到开发环境
	helm install asynqhub deployments/helm/asynq-hub -f deployments/helm/asynq-hub/values.yaml --namespace asynqhub-dev --create-namespace

helm-upgrade: ## 使用 Helm 升级
	helm upgrade asynqhub deployments/helm/asynq-hub -f deployments/helm/asynq-hub/values.yaml --namespace asynqhub-dev

helm-uninstall: ## 卸载 Helm release
	helm uninstall asynqhub --namespace asynqhub-dev

tag: ## 创建 Git 标签
	@read -p "Enter tag version (e.g., v1.0.0): " tag; \
	git tag -a $$tag -m "Release $$tag"; \
	git push origin $$tag
