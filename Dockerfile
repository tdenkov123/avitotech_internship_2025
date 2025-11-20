# syntax=docker/dockerfile:1.7

FROM golang:1.24 AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/app ./cmd/app

FROM alpine:3.22
WORKDIR /app
RUN adduser -D -g '' appuser
COPY --from=builder /bin/app ./service
EXPOSE 8080
USER appuser
ENTRYPOINT ["/app/service"]
