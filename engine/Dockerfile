FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/engine cmd/engine/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/bitcode cmd/bitcode/main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/bin/engine /app/engine
COPY --from=builder /app/bin/bitcode /app/bitcode
COPY --from=builder /app/modules /app/modules

EXPOSE 8080
CMD ["/app/engine"]
