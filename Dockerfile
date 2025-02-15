FROM golang:1.23.4-alpine AS builder

RUN apk update && apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
COPY internal ./internal

RUN CGO_ENABLED=1 GOOS=linux go build -o /wishbot

FROM alpine:latest

WORKDIR /app

COPY --from=builder /wishbot .

CMD ["/app/wishbot"]
