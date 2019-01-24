

build-docker:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
	docker build -t xetys/k8s-bitflow-operator .
