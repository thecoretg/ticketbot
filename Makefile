create-bin-dir:
	mkdir -p bin

build-cli: create-bin-dir
	go build -o bin/cli ./cmd/cli && cp bin/cli ~/go/bin/tbot

gensql:
	sqlc generate

runserver: test-db-down test-db-up
	op run --env-file="./testing.env" --no-masking -- go run ./cmd/server

create-test-rule: build-cli
	op run --env-file="./testing.env" --no-masking -- tbot sync -r -b -t -i 38
	sleep 10
	op run --env-file="./testing.env" --no-masking -- tbot create rule -b 38 -r 1

test-db-up:
	docker compose -f ./docker/docker-compose-db.yml up -d

test-db-down:
	docker compose -f ./docker/docker-compose-db.yml down -v

docker-build:
	docker buildx build --platform=linux/amd64 -t ticketbot:v1.3.1 --load -f ./docker/DockerfileMain .

deploy-lightsail: docker-build
	aws lightsail push-container-image \
	--region us-west-2 \
	--service-name ticketbot \
	--label ticketbot-server \
	--image ticketbot:v1.3.1

lightsail-logs:
	aws lightsail get-container-log \
	--service-name ticketbot \
	--container-name ticketbot \
	--output text
