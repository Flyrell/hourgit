FROM golang:1.26-alpine AS builder

ARG VERSION=dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /build/hourgit \
    ./cmd/hourgit

FROM scratch

COPY --from=builder /build/hourgit /hourgit

ENTRYPOINT ["/hourgit"]
