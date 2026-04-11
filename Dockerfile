FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o horizon-exporter .

FROM scratch
COPY --from=builder /app/horizon-exporter /horizon-exporter
EXPOSE 9888
ENTRYPOINT ["/horizon-exporter"]
