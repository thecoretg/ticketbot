VERSION  ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
# App env file. A 1Password-mounted .env at the repo root is the expected setup; override with
# `make run ENV_FILE=path/to/.env`.
ENV_FILE ?= .env
# 1Password Environment ("Ticketbot Testing") that `make run` loads through the op CLI.
OP_ENVIRONMENT ?= hazkhine3rro7guhx3en2dukqq
COMPOSE   = docker compose

gensql:
	sqlc generate

lint:
	golangci-lint run ./...

# ── Local development: Postgres in Docker, Go on the host ────────────────────
db-up:
	$(COMPOSE) up -d db

db-down:
	$(COMPOSE) stop db

# Removes the database volume too. Only when you really want a fresh DB.
db-nuke:
	$(COMPOSE) down -v

# With OP_ENVIRONMENT set, the 1Password CLI injects that Environment's variables directly: the
# mounted .env pipe serves one read and then empty ones, so reading it is unreliable. Set it
# empty (`make run OP_ENVIRONMENT=`) to load ENV_FILE instead. That file is read line by line
# rather than sourced: macOS /bin/sh (bash 3.2) sources a 1Password-mounted file as empty.
run:
ifneq ($(OP_ENVIRONMENT),)
	op run --environment $(OP_ENVIRONMENT) -- go run .
else
	@test -e $(ENV_FILE) || { echo "missing $(ENV_FILE); see .env.example"; exit 1; }
	@set -a; while IFS= read -r l; do case "$$l" in ""|"#"*) ;; *) export "$$l";; esac; done < $(ENV_FILE); set +a; go run .
endif

# ── Full stack in Docker ─────────────────────────────────────────────────────
docker-up:
	VERSION=$(VERSION) $(COMPOSE) --env-file $(ENV_FILE) up --build

docker-down:
	$(COMPOSE) --env-file $(ENV_FILE) down

docker-build:
	docker buildx build --platform=linux/amd64 --build-arg VERSION=$(VERSION) -t ticketbot:$(VERSION) --load .

deploy-container: docker-build
	aws lightsail push-container-image \
	--region us-west-2 \
	--service-name ticketbot \
	--label ticketbot-server \
	--image ticketbot:$(VERSION)
