package main

import (
	"context"
	"os"
	"os/signal"
	"scheduler-service/internal/application"
	"syscall"
	// Embedded zone data: hub opening hours are in IANA zones, and the slim
	// runtime image has no /usr/share/zoneinfo.
	_ "time/tzdata"
)

func main() {
	// Cancelled on SIGINT (Ctrl+C) or SIGTERM (docker stop / k8s pod
	// termination), which triggers the application's graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application.SchedulerApplication(ctx)
}
