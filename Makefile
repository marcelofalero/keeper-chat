.PHONY: init up down status pristine help all

init: up ## Initialize and run database migrations
	@echo "Waiting for PostgreSQL to be healthy..."
	@bash -c ' \
		while ! docker-compose exec postgresd pg_isready -U hydra -d hydra -q; do \
			echo -n "."; \
			sleep 1; \
		done; \
		echo "PostgreSQL is ready."; \
	'
	@echo "Running Kratos migrations..."
	docker-compose exec kratos kratos migrate sql -e -c /etc/config/kratos/kratos.yml --yes
	@echo "Running Keto migrations..."
	docker-compose exec keto keto migrate up -c /etc/config/keto/keto.yml --yes
	@echo "Initialization and migrations complete."

up: ## Start all services in detached mode
	docker-compose up -d

down: ## Stop and remove all services, networks, and volumes
	docker-compose down -v

status: ## Display the status of the Docker Compose services
	docker-compose ps

pristine: ## Stop services, remove volumes, and all images used by services
	@echo "Stopping services, removing volumes, and removing images used by services..."
	docker-compose down -v --rmi all
	@echo "System cleaned to pristine state (excluding .env and config files)."

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Default target
all: up

# Add .DEFAULT_GOAL to ensure 'help' is not the default
.DEFAULT_GOAL := all
