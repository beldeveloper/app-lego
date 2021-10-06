package main

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/controller"
	"github.com/beldeveloper/app-lego/model"
	"github.com/beldeveloper/app-lego/provider/rest"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	c, err := InitializeController()
	if err != nil {
		log.Fatalf("main: %v\n", err)
	}
	ctx := context.Background()
	go c.DownloadRepositoryJob(ctx)
	go c.SyncRepositoryJob(ctx)
	go c.BuildBranchJob(ctx)
	go c.WatchDeploymentsJob(ctx)
	runHttpServer(c)
}

func postgresConn() (*pgxpool.Pool, error) {
	pgs := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("APP_LEGO_DB_HOST"),
		os.Getenv("APP_LEGO_DB_PORT"),
		os.Getenv("APP_LEGO_DB_USER"),
		os.Getenv("APP_LEGO_DB_PASSWORD"),
		os.Getenv("APP_LEGO_DB_NAME"),
	)
	return pgxpool.Connect(context.Background(), pgs)
}

func postgresSchema() model.PgSchema {
	return model.PgSchema(os.Getenv("APP_LEGO_DB_SCHEMA"))
}

func workDir() model.FilePath {
	return model.FilePath(strings.TrimRight(os.Getenv("APP_LEGO_WORKING_DIR"), "/"))
}

func runHttpServer(c controller.Service) {
	httpPort := os.Getenv("APP_LEGO_HTTP_PORT")
	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: rest.CreateRouter(c),
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("main: serve http: %v; port = %s\n", err, httpPort)
		}
	}()
	log.Printf("Listening :%s for HTTP connections...\n", httpPort)
	<-done
	log.Print("Stopping the application...\n")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("main: server shutdown: %v\n", err)
	}
}
