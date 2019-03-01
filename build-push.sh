#!/bin/sh

CONTAINER_IMAGE=$(cat deploy/operator.yaml| grep image | sed -e 's/image: //g' | xargs)

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

docker build -t ${CONTAINER_IMAGE} .
docker push ${CONTAINER_IMAGE}