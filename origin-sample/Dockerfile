FROM golang:1.25 AS builder

WORKDIR /app

# Copy only this service
COPY origin-sample/ ./

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o app main.go

FROM alpine:3.19
WORKDIR /root/
COPY --from=builder /app/app .
CMD ["./app"]
