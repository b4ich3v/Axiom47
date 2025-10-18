# XDP47 Control (Ubuntu-based) - multi-stage
FROM golang:1.23-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/xdp47-control ./cmd/xdp47-control

FROM ubuntu:22.04
RUN useradd -m -u 10001 app
WORKDIR /app
COPY --from=build /out/xdp47-control /usr/local/bin/xdp47-control
ENV XDP47_LISTEN_ADDR=":8080"
EXPOSE 8080
USER app
ENTRYPOINT ["xdp47-control"]
