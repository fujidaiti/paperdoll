package worker

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/feed"
	"github.com/fujidaiti/paperdoll/server/feature/newspaper"
	"github.com/fujidaiti/paperdoll/server/feature/scraper"
	"github.com/go-co-op/gocron/v2"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func StartScheduler(ctx context.Context) {
	dsn := os.Getenv("DB_DSN")
	if len(dsn) == 0 {
		panic("DB_DSN is required.")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		panic(err)
	}
	defer func() {
		fmt.Println("Closing DB...")
		if err := db.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	if err := db.Ping(); err != nil {
		panic(err)
	}

	newspaperSvc := &newspaper.Service{DB: db}
	scraperSvc := scraper.NewService(nil)

	p := &pool{}
	p.start(ctx, 16)
	defer func() {
		fmt.Println("Shutting down workers...")
		p.shutdown()
	}()

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		panic(err)
	}
	defer func() {
		fmt.Println("Shutting down scheduler...")
		if err := scheduler.Shutdown(); err != nil {
			fmt.Println(err)
		}
	}()
	_, err = scheduler.NewJob(
		gocron.DurationJob(time.Hour),
		gocron.NewTask(func() {
			fmt.Println("--------------------------")
			fmt.Println("Rfreshing feeds...")
			jobs, err := feed.CollectJobs(ctx, db, newspaperSvc, scraperSvc, time.Now())
			if err != nil {
				fmt.Println(err)
				return
			}
			if len(jobs) == 0 {
				fmt.Println("No polling is scheduled.")
				return
			}
			for _, job := range jobs {
				err := p.push(ctx, &job)
				if err != nil {
					fmt.Println(err)
					return
				}
			}
		}),
	)
	if err != nil {
		panic(err)
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(time.Hour),
		gocron.NewTask(func() {
			n, err := feed.DeleteExpiredSelectorEditors(ctx, db)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Printf("Deleted %d expired selector editors.\n", n)
		}),
	)
	if err != nil {
		panic(err)
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(5*time.Minute),
		gocron.NewTask(func() {
			jobs, err := newspaper.CollectJobs(ctx, db, time.Now())
			if err != nil {
				fmt.Println(err)
			}
			for _, job := range jobs {
				err := p.push(ctx, &job)
				if err != nil {
					fmt.Println(err)
					return
				}
			}
		}),
	)
	if err != nil {
		panic(err)
	}
	scheduler.Start()
	<-ctx.Done()
}
