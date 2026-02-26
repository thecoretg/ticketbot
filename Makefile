create-bin-dir:
	mkdir -p bin

build-server: create-bin-dir
	go build -o bin/server ./cmd/server

build-cli: create-bin-dir
	go build -o bin/tbot-admin ./cmd/tbot-admin && cp bin/tbot-admin ~/go/bin/tbot-admin

tui:
	op run --env-file="./testing.env" --no-masking -- go run ./cmd/tbot-admin

gensql:
	sqlc generate

runserver:
	op run --env-file="./testing.env" --no-masking -- go run ./cmd/server

sync:
	op run --env-file="./testing.env" --no-masking -- tbot-admin sync -b -r

test-db-up:
	docker compose -f ./docker/docker-compose-db.yml up -d

test-db-down:
	docker compose -f ./docker/docker-compose-db.yml down -v

reset-test-db: test-db-down test-db-up

docker-build:
	docker buildx build --platform=linux/amd64 -t ticketbot:v1.4.2 --load -f ./docker/DockerfileMain .

deploy-container: docker-build
	aws lightsail push-container-image \
	--region us-west-2 \
	--service-name ticketbot \
	--label ticketbot-server \
	--image ticketbot:v1.4.2

lightsail-logs:
	aws lightsail get-container-log \
	--service-name ticketbot \
	--container-name ticketbot \
	--output text
