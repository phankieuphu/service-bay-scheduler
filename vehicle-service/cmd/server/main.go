package main

import (
	"context"
	"vehicle-service/internal/application"
)

func main() {
	ctx := context.Background()
	application.VehicleApplication(ctx)

}
