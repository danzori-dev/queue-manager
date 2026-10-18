<div align="center">

# Queue Manager

Telegram bot for managing queues in a university study group.

[![CI](https://github.com/danzori-dev/queue-manager/actions/workflows/ci.yml/badge.svg)](https://github.com/danzori-dev/queue-manager/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/danzori-dev/queue-manager)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/danzori-dev/queue-manager)](https://goreportcard.com/report/github.com/danzori-dev/queue-manager)
[![Release](https://img.shields.io/github/v/release/danzori-dev/queue-manager)](https://github.com/danzori-dev/queue-manager/releases)

</div>

## Run

```bash
cp .env.example .env # fill in the values
VERSION=$(git describe --tags --always --dirty) docker compose up -d --build
```

## Development

```bash
task migrate:up
go run ./cmd/queuemanager
```