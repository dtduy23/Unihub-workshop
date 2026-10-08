# ==============================================================================
# UNIHUB WORKSHOP — DEVOPS AUTOMATION MAKEFILE
# ==============================================================================

.PHONY: help dev dev-down dev-logs build test bench bench-2k k8s-deploy clean

help: ## Hiển thị danh sách các lệnh hỗ trợ
	@echo "UniHub Platform — Command Reference:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# --- LOCAL DEVELOPMENT (DOCKER COMPOSE) ---
dev: ## Khởi động môi trường 16-replica container local (Postgres, Redis, RabbitMQ, API, Worker, Web)
	docker compose up -d

dev-down: ## Dừng và gỡ bỏ toàn bộ container local
	docker compose down

dev-logs: ## Xem realtime logs của toàn bộ hệ thống
	docker compose logs -f

dev-ps: ## Kiểm tra trạng thái các container
	docker compose ps

# --- BUILD & TEST ---
build: ## Biên dịch Go Backend & công cụ Concurrency Demo
	@echo ">>> Building backend & demo tools..."
	@mkdir -p src/backend/bin
	(cd src/backend && go build -o bin/server ./cmd/server)
	(cd src/backend && go build -o bin/concurrency_demo ./cmd/concurrency_demo)
	@echo ">>> Done. Binaries saved in src/backend/bin/"

test: ## Chạy Unit Tests kèm bộ kiểm tra Data Race (-race)
	@echo ">>> Running Go tests with Data Race detection..."
	(cd src/backend && go test -v -race ./...)

# --- LOAD TEST & BENCHMARK ---
bench: ## Chạy nhanh Stress-test (50 users, 25 slots)
	@./scripts/benchmark/run_concurrency_test.sh 50 25 1 gate

bench-2k: ## Chạy Full Gate Burst Stress-test (2,000 users, 100 slots, 0ms barrier)
	@./scripts/benchmark/run_concurrency_test.sh 2000 100 1 gate

# --- KUBERNETES & GITOPS ---
k8s-deploy: ## Tự động hóa deploy lên cụm Minikube / Local K8s
	@bash scripts/devops/bootstrap-demo.sh minikube

clean: ## Dọn dẹp các file binary đã build
	rm -rf src/backend/bin/*

# --- COMPLETE DEVOPS ARTIFACTS ---
.PHONY: devops-check devops-demo devops-demo-kind devops-ci devops-loadtest

devops-check: ## Kiểm tra Helm, GitOps, CI DAG, shell và Python
	@bash scripts/devops/validate.sh

devops-demo: ## Dựng stack DevOps + GitOps snapshot trên minikube riêng
	@bash scripts/devops/bootstrap-demo.sh minikube

devops-demo-kind: ## Dựng stack DevOps + GitOps snapshot trên kind riêng
	@bash scripts/devops/bootstrap-demo.sh kind

devops-ci: ## Submit Argo CI (yêu cầu IMAGE_REGISTRY và registry credentials)
	@bash scripts/devops/submit-ci.sh

devops-loadtest: ## Chạy k6 gate trên namespace demo (yêu cầu BACKEND_IMAGE)
	@bash scripts/devops/run-loadtest.sh unihub-demo "$(BACKEND_IMAGE)"
