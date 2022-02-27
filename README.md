# gota
Arduino http OTA server implemented in golang

It's a very simple example. Do not use for anything :)

## how to use

upload file with curl to this server

`curl -X POST -F "file=@main.go" localhost:8080/upload/`

download file with curl from this server

`curl http://localhost:8080/download/main.go`