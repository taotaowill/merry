#!/bin/bash

go mod tidy
go build -o cli client/client.go
go build -o ser server/server.go

