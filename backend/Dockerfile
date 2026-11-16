FROM golang:1.26-bookworm

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /usr/local/bin/booking ./cmd/booking

EXPOSE 8080
