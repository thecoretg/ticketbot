VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")

.PHONY: gensql vendor docker-up docker-down docker-build deploy-container

gensql:
	sqlc generate

vendor:
	go mod tidy
	go mod vendor

docker-up: vendor
	docker compose -f ./docker/docker-compose.yml up --build

docker-down:
	docker-compose -f ./docker/docker-compose.yml down -v

docker-build: vendor
	docker buildx build --platform=linux/amd64 -t ticketbot:$(VERSION) --load -f ./docker/DockerfileMain .

deploy-container: docker-build
	aws lightsail push-container-image \
	--region us-west-2 \
	--service-name ticketbot \
	--label ticketbot-server \
	--image ticketbot:$(VERSION)
