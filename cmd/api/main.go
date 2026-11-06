package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mgmd-abdulaev/bazgain-places-go/internal/app"
	"github.com/mgmd-abdulaev/bazgain-places-go/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "configuration error:", err)
		os.Exit(1)
	}
	if err := app.Run(context.Background(), cfg); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "service stopped:", err)
		os.Exit(1)
	}
}
