FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go env -w GOSUMDB=off && go mod tidy
RUN go build -o /bin/api ./cmd/api
RUN go build -o /bin/worker ./cmd/worker

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /bin/api .
COPY --from=builder /bin/worker .
CMD ["./api"]
