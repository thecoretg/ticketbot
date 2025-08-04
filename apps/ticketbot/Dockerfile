FROM golang:1.24.2 AS builder
WORKDIR /app
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o ticketbot ./apps/ticketbot

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/ticketbot .
CMD ["./ticketbot"]