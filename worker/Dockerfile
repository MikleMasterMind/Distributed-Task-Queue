FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY worker/go.mod worker/go.sum ./
RUN go mod download

COPY worker/ .
RUN CGO_ENABLED=0 go build -o /worker ./cmd/worker

FROM alpine:3.20

COPY --from=builder /worker /worker

ENTRYPOINT ["/worker"]
