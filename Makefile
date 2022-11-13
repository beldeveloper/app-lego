APP=app-lego
GOOS?=linux

build:
	CGO_ENABLED=0 GOOS=${GOOS} go build -o ${APP} ./cmd/server/.
