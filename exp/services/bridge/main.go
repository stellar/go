package main

import (
	"context"
	"github.com/go-pg/pg/v10"
	"github.com/gofiber/fiber/v2"
	"os"
	"strconv"
	"strings"
)

var domain string
var account string
var db *pg.DB

func main() {
	domain = os.Getenv("DOMAIN")
	if domain == "" {
		domain = "localhost"
	}

	account = os.Getenv("ACCOUNT")

	db = pg.Connect(&pg.Options{Database: "bridge"})

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		panic(err)
	}

	app := fiber.New()

	app.Post("/create-virtual-account", CreateVirtualAccount)
	app.Get("/federation", Federation)
	app.Post("/send-to-stellar", SendToStellar)
	app.Get("/payments-from-stellar", PaymentsFromStellar)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	app.Listen(":" + port)
}
