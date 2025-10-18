# XDP47 Agent (Ubuntu-based) - multi-stage
FROM golang:1.23-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/xdp47-agent ./cmd/xdp47-agent

FROM ubuntu:22.04
RUN useradd -m -u 10001 app
WORKDIR /app
COPY --from=build /out/xdp47-agent /usr/local/bin/xdp47-agent
ENV XDP47_CONTROL_URL="http://control:8080"
ENV XDP47_TENANT="demo-tenant"
ENV XDP47_DEVICE_LABELS="store=sofia-01,role=kiosk"
USER app
ENTRYPOINT ["xdp47-agent"]
