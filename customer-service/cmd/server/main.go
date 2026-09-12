package main

import (
	"context"
	"customer-service/internal/application"
)

func main() {
	ctx := context.Background()
	application.CustomerApplication(ctx)

}
