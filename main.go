package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"events-system/api"
	"events-system/database"
)

func main() {
	// Get database DSN from environment variable
	dbDSN := os.Getenv("POSTGRES_DSN")
	if dbDSN == "" {
		dbDSN = "postgres://postgres:postgres@localhost:5432/eventsdb?sslmode=disable"
		log.Println("using default database DSN")
	} else {
		log.Printf("connecting to database using POSTGRES_DSN from environment")
	}

	log.Printf("attempting to connect to database...")
	// Initialize database connection
	db, err := database.Connect(dbDSN)
	if err != nil {
		log.Fatal("database connect:", err)
	}
	log.Println("successfully connected to database")
	defer db.Close()

	service := api.NewAPI(db)
	service.RegisterRoutes()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("server starting on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), service.Handler()))
}
