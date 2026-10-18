package main

import (
	"context"
	"os"

	"github.com/danzori-dev/queue-manager/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(context.Background()); err != nil {
		os.Exit(1)
	}
}
