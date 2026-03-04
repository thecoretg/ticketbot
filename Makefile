gensql:
	sqlc generate

docker-up:
	docker compose -f ./docker/docker-compose.yml up --build

docker-down:
	docker-compose -f ./docker/docker-compose.yml down -v

docker-build:
	docker buildx build --platform=linux/amd64 -t ticketbot:v1.5.1 --load -f ./docker/DockerfileMain .

deploy-container: docker-build
	aws lightsail push-container-image \
	--region us-west-2 \
	--service-name ticketbot \
	--label ticketbot-server \
	--image ticketbot:v1.5.1
