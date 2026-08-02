FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go env -w GOSUMDB=off && go mod tidy
RUN go build -o /bin/api ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /bin/api .
CMD ["./api"]
