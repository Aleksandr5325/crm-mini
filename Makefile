include .env
export

export PROJECT_ROOT=${shell pwd}

.DEFAULT_GOAL=help

env-up: ## Запустить окружение проекта
	@docker compose up -d crm-postgres

env-down: ## Остановить окружение проекта
	@docker compose down crm-postgres

env-cleanup: ## Отчистить окружение проекта
	@read -p "Очистить все volume файлы окружения? Опасность утери данных. [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then \
		docker compose down crm-postgres port-forwarder && \
		rm -rf ${PROJECT_ROOT}/out/pgdata && \
		echo "Файлы окружения очищены"; \
	else \
		echo "Очистка окружения отменена"; \
	fi

env-port-forwarder: ## Открыть порты соединения
	@docker compose up -d port-forwarder

env-port-close: ## Закрыть порты соединения
	@docker compose down port-forwarder

ps: ## Посмотреть активные сервисы
	@docker compose ps

help: ## Показать справку по командам 
	@echo "=== Центр управления проектом ==="
	@echo ""
	@echo "Доступные команды:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

migrate-create: ## PostgreSQL: Создать новую версию схемы данных
	@if [ -z "$(seq)" ]; then \
		echo "Отсутствует необходимый параметр seq. Пример: make migrate-create seq=init"; \
		exit 1; \
	fi; \
	docker compose run --rm crm-postgres-migrate \
		create \
		-ext sql \
		-dir /migrations \
		-seq "$(seq)"

migrate-up: ## PostgreSQL: Накатить миграции
	@make migrate-action action=up

migrate-down: ## PostgreSQL: Откатить миграции
	@make migrate-action action=down

migrate-action:
	@if [ -z "$(action)" ]; then \
		echo "Отсутствует необходимый параметр action. Пример: make migrate-action action=up"; \
		exit 1; \
	fi; \
	docker compose run --rm crm-postgres-migrate \
		-path /migrations \
		-database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@crm-postgres:5432/${POSTGRES_DB}?sslmode=disable&search_path=public" \
		"$(action)"

crm-run: ## Go-приложение: запустить локально.
	@POSTGRES_HOST=127.0.0.1 HTTP_ADDR=127.0.0.1:8080 go run ./cmd/crm

crm-deploy: ## Go-приложение: собрать образ и запустить всё в Docker
	@docker compose up -d --build crm

crm-undeploy: ## Go-приложение: остановить контейнер приложения
	@docker compose stop crm