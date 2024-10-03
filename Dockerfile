FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main ./cmd/web

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
COPY web ./web
EXPOSE 8080
CMD ["./main"]