FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/collector-backend .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=builder /out/collector-backend /app/collector-backend

EXPOSE 8080
ENTRYPOINT ["/app/collector-backend"]
