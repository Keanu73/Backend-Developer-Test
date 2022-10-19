package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5"
)

var db *pgx.Conn
var err error

// Default database settings
var (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "password"
	dbname   = "spots"
)

// Connect function
func Connect(ctx context.Context) error {
	pgPort, ok := os.LookupEnv("PG_PORT")
	if ok {
		port, err = strconv.Atoi(pgPort)
		if err != nil {
			return err
		}
	}

	pgHost, ok := os.LookupEnv("PG_HOST")
	if ok {
		host = pgHost
	}

	db, err = pgx.Connect(
		ctx, fmt.Sprintf("postgres://%s:%s@%s:%d/%s", user, password, host, port, dbname),
	)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}
