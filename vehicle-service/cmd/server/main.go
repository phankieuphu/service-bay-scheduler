package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"vehicle-service/internal/application"
)

func main() {
	// Cancelled on SIGINT (Ctrl+C) or SIGTERM (docker stop / k8s pod
	// termination), which triggers the application's graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application.VehicleApplication(ctx)
}
