package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN is required")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	for i := 0; i < 10; i++ {
		if err := db.Ping(); err == nil {
			log.Println("database connected")
			break
		}
		log.Printf("db not ready, retrying... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}

	app := NewApp(NewStore(db))

	port := strings.TrimSpace(os.Getenv("APP_PORT"))
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, app.Routes()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
