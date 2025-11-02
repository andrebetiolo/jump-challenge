# Load enviroment variables
-include .env

# Export enviroment variables to commands
export

# Variables
go_cover_file=coverage.out

help:: ## Show this help
	@ fgrep -h "##" $(MAKEFILE_LIST) | sort | fgrep -v fgrep | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

dev:: ## Run go Application with watcher
	@ go run cmd/server/main.go

test:: ## Do the tests in go
	@ go test -race -coverprofile $(go_cover_file) ./...

kill:: ## Kill debug process
	@ kill -9 $(shell lsof -ti:$(PORT))

cover:: test ## See coverage of tests, see more in https://go.dev/blog/cover
	@ go tool cover -func=$(go_cover_file)

cover-html:: test ## See of the coverage of the code on your default navigator
	@ go tool cover -html=$(go_cover_file)
