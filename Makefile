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
