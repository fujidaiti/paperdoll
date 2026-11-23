package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/feed"
	"github.com/fujidaiti/paperdoll/server/feature/scraper"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Specify a sub-command.")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "poll":
		poll()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

// poll fetches feeds and content immediately.
func poll() {
	ctx := context.Background()
	db, err := setUpDB()
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	jobs, err := feed.CollectJobs(ctx, db, scraper.NewService(nil), time.Now())
	if err != nil {
		panic(err)
	}
	if len(jobs) == 0 {
		fmt.Println("No feed found to poll. Skipping.")
		return
	}
	for _, job := range jobs {
		err := job.Do(ctx)
		if err != nil {
			fmt.Print(err)
		}
	}
}

func setUpDB() (*sql.DB, error) {
	dsn := os.Getenv("DB_DSN")
	if len(dsn) == 0 {
		return nil, fmt.Errorf("DB_DSN is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		if err2 := db.Close(); err2 != nil {
			fmt.Println(err2)
		}
		return nil, err
	}
	return db, nil
}
