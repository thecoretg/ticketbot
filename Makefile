update-lambda:
	scripts/deploy_lambda.sh

gensql:
	sqlc generate -f db/sqlc.yaml

init-hooks:
	go run cmd/cli/main.go init-hooks

preload-db:
	go run cmd/cli/main.go preload -b -t 15

runserver:
	go run cmd/cli/main.go run

up:
	goose up

down:
	goose down