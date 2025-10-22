create-bin-dir:
	mkdir -p bin

build-server: create-bin-dir
	go build -o bin/server ./cmd/server && sudo cp bin/server /usr/local/bin/tbot-server

build-admin-cli: create-bin-dir
	go build -o bin/cli ./cmd/admincli && sudo cp bin/admincli /usr/local/bin/tbot-cli

gensql:
	sqlc generate -f internal/db/sqlc.yaml

up:
	goose up

down:
	goose down