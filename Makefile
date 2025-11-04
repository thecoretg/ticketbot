create-bin-dir:
	mkdir -p bin

build: create-bin-dir
	go build -o bin/server ./cmd/server && sudo cp bin/server /usr/local/bin/tbot-server

gensql:
	sqlc generate

docker-build:
	docker buildx build --platform=linux/amd64 -t ticketbot:latest --load .

test-up:
	docker compose up --build

test-down:
	docker compose down -v

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