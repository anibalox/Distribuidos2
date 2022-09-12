# Ejemplo

## Version local o una sola VM

### Compilar proto
    protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative Proto/message.proto

### GO

Ejecutar cada uno en una consola distinta.

Para la central.
    
    go run Central/main.go

Para el Laboratorio

    go run Laboratorio/main.go
