FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o gosendmail main.go

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

RUN adduser -D -u 1000 appuser

WORKDIR /home/appuser

COPY --from=builder /app/gosendmail .

USER appuser

EXPOSE 8080

CMD ["./gosendmail"]
