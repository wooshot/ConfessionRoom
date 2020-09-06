# Confession Room
## prerequire:
```
  golang 1.15
  docker: 19.03
  protobuf: 3.7.1
```

## modules:
    go mod vendor
## generate pb file
    make proto
## Run server (local)
    go run main.go
## RUN server (docker)
    docker-compose up --build

```
chat api : {host}:8000
restful api : {host}:8090
grpc api : {host}:8091
swagger: {host}:8092
```
