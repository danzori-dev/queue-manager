ARG GO_VERSION=1.27
ARG ALPINE_VERSION=3.24
ARG GOOSE_VERSION=v3.29.0

FROM golang:${GO_VERSION}-alpine AS build

ARG VERSION

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN VERSION="${VERSION:-$(git describe --tags --always)}" \
    && CGO_ENABLED=0 go build -trimpath -buildvcs=false \
    -ldflags="-s -w -X github.com/danzori-dev/queue-manager/internal/bootstrap.buildVersion=${VERSION}" \
    -o /out/queuemanager ./cmd/queuemanager

FROM golang:${GO_VERSION}-alpine AS goose

ARG GOOSE_VERSION

RUN CGO_ENABLED=0 GOBIN=/out go install -trimpath -ldflags="-s -w" \
    github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION}

FROM alpine:${ALPINE_VERSION} AS migrate

RUN apk add --no-cache ca-certificates \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=goose /out/goose /usr/local/bin/goose
COPY migrations/*.sql /migrations/

USER appuser

ENTRYPOINT ["/bin/sh", "-c", "exec goose -dir /migrations turso \"$SQLITE_DSN?authToken=$SQLITE_AUTH_TOKEN\" \"${1:-up}\"", "--"]

FROM alpine:${ALPINE_VERSION} AS runtime

RUN apk add --no-cache ca-certificates \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /src

COPY --from=build --chown=appuser:appgroup /out/queuemanager /usr/local/bin/queuemanager
COPY --chown=appuser:appgroup configs/config.yml /src/configs/config.yml

USER appuser

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/queuemanager"]