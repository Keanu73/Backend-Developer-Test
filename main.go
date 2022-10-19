package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := Bootstrap()

	log.Fatal(app.Listen(":3000"))
}

func Bootstrap() *fiber.App {
	ctx := context.Background()

	// Connect with database
	if err := Connect(ctx); err != nil {
		log.Fatal(err)
	}

	// Create a Fiber app
	app := fiber.New()

	// Assigns routes to handler functions
	app.Get("/spots/duplicates", DuplicateSpots(ctx))
	app.Get("/spots/inArea", SpotsInArea(ctx))

	return app
}
