FROM golang:1.26.1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/i-dra-driver cmd/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/bin/i-dra-driver .

ENTRYPOINT ["./i-dra-driver"]
