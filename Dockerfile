FROM golang:1.24.4 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION
ARG COMMIT
ARG BUILT_AT

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o app ./cmd/vk-sender

FROM gcr.io/distroless/static:nonroot
WORKDIR /app

COPY --from=builder /app/app /app/app

USER nonroot:nonroot
ENTRYPOINT ["/app/app"]
