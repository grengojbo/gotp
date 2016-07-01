BIN_NAME=print-pos

CUR_TIME=$(shell date '+%Y-%m-%d_%H:%M:%S')
# Program version
VERSION=$(shell cat VERSION)

# Grab the current commit
GIT_COMMIT="$(shell git rev-parse HEAD)"

all: arm cli

arm: clean
	@mkdir -p ./dist-arm
	@GOOS=linux GOARCH=arm GOARM=7 go build -a -tags 'linux netgo' -o dist-arm/${BIN_NAME} main.go

clean:
	@test ! -e ./${BIN_NAME} || rm ./${BIN_NAME}
	@test ! -e ./dist-arm/${BIN_NAME} || rm ./dist-arm/${BIN_NAME}
	@git gc --prune=0 --aggressive
	@find . -name "*.orig" -type f -delete
	@find . -name "*.log" -type f -delete

cli: 
	@echo "Building cli ${VERSION}"
	@go build -a -tags netgo -ldflags '-w -X cmd.BuildTime=${CUR_TIME} -X cmd.Version=${VERSION} -X cmd.GitHash=${GIT_COMMIT}' -o $(BIN_NAME) main.go
	@chmod 0755 ./$(BIN_NAME)
