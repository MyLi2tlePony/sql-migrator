FROM golang:1.18.3 as build

ENV BIN_FILE /bin/app
ENV CODE_DIR /go/src/

# Кэшируем слои с модулями
WORKDIR ${CODE_DIR}
COPY go.mod .
COPY go.sum .
RUN go mod download

# Копипуем код
COPY . ${CODE_DIR}

CMD go test -tags=integration ./...