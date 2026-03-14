COMPOSE := docker compose

.PHONY: test test-e2e compose-up compose-down compose-logs compose-ps compose-build compose-rebuild compose-clean-images compose-reset migrate-up migrate-down

test:
	cd backend && go test ./...

test-e2e:
	python3 backend/scripts/e2e_api_regression.py

restart:
	$(COMPOSE) down && $(COMPOSE) up -d
	$(MAKE) migrate-up

compose-up:
	$(COMPOSE) up -d
	$(MAKE) migrate-up

compose-down:
	$(COMPOSE) down

compose-logs:
	$(COMPOSE) logs -f

compose-ps:
	$(COMPOSE) ps

compose-build:
	$(COMPOSE) build

compose-rebuild:
	$(COMPOSE) down
	$(COMPOSE) build
	$(MAKE) compose-up

compose-clean-images:
	$(COMPOSE) down --rmi all --remove-orphans

compose-reset:
	$(COMPOSE) down --rmi all --volumes --remove-orphans
	$(COMPOSE) up -d --build
	$(MAKE) migrate-up

migrate-up:
	$(COMPOSE) --profile tools run --rm migrate

migrate-down:
	$(COMPOSE) --profile tools run --rm migrate -path /migrations -database "$${DATABASE_URL:-postgresql://interviewly:interviewly@postgres:5432/interviewly?sslmode=disable}" down 1
