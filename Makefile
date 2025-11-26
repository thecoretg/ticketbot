create-bin-dir:
	mkdir -p bin

build-cli: create-bin-dir
	go build -o bin/cli ./cmd/cli && sudo cp bin/cli /usr/local/bin/tbot-admin

gensql:
	sqlc generate

runserver:
	go run ./cmd/server

docker-build:
	docker buildx build --platform=linux/amd64 -t ticketbot:latest --load .

test-db-up:
	docker compose -f docker-compose-db.yml up -d

test-db-down:
	docker compose -f docker-compose-db.yml down -v

test-api-up:
	docker compose -f docker-compose-api.yml up --build

test-api-down:
	docker compose -f docker-compose-api.yml down -v

deploy-lightsail: docker-build
	aws lightsail push-container-image \
	--region us-west-2 \
	--service-name ticketbot \
	--label ticketbot-server \
	--image ticketbot:latest

lightsail-logs:
	aws lightsail get-container-log \
	--service-name ticketbot \
	--container-name ticketbot \
	--output text