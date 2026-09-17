VERSION  ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
# App env file. A 1Password-mounted .env at the repo root is the expected setup; override with
# `make run ENV_FILE=path/to/.env`.
ENV_FILE ?= .env
COMPOSE   = docker compose -f ./docker/docker-compose.yml

gensql:
	sqlc generate

# ── Local development: Postgres in Docker, Go on the host ────────────────────
db-up:
	$(COMPOSE) up -d db

db-down:
	$(COMPOSE) stop db

# Removes the database volume too. Only when you really want a fresh DB.
db-nuke:
	$(COMPOSE) down -v

# The env file is read line by line instead of sourced: macOS /bin/sh (bash 3.2) sources a
# 1Password-mounted file as empty.
run:
	@test -e $(ENV_FILE) || { echo "missing $(ENV_FILE); see .env.example"; exit 1; }
	@set -a; while IFS= read -r l; do case "$$l" in ""|"#"*) ;; *) export "$$l";; esac; done < $(ENV_FILE); set +a; go run .

# ── Full stack in Docker ─────────────────────────────────────────────────────
docker-up:
	$(COMPOSE) --env-file $(ENV_FILE) up --build

docker-down:
	$(COMPOSE) --env-file $(ENV_FILE) down

docker-build:
	docker buildx build --platform=linux/amd64 -t ticketbot:$(VERSION) --load -f ./docker/DockerfileMain .

deploy-container: docker-build
	aws lightsail push-container-image \
	--region us-west-2 \
	--service-name ticketbot \
	--label ticketbot-server \
	--image ticketbot:$(VERSION)
