package main

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/controller"
	"github.com/beldeveloper/app-lego/provider/rest"
	"github.com/beldeveloper/app-lego/service"
	"github.com/beldeveloper/app-lego/service/branch"
	"github.com/beldeveloper/app-lego/service/builder"
	"github.com/beldeveloper/app-lego/service/deployer"
	"github.com/beldeveloper/app-lego/service/deployment"
	"github.com/beldeveloper/app-lego/service/marshaller"
	appOs "github.com/beldeveloper/app-lego/service/os"
	"github.com/beldeveloper/app-lego/service/repository"
	"github.com/beldeveloper/app-lego/service/validation"
	"github.com/beldeveloper/app-lego/service/variable"
	"github.com/beldeveloper/app-lego/service/vcs"
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
	ctx := context.Background()
	pgConn, err := postgresConn(ctx)
	pgSchema := os.Getenv("APP_LEGO_DB_SCHEMA")
	if err != nil {
		log.Fatalf("main: establish postgress connection: %v\n", err)
	}
	workDir := strings.TrimRight(os.Getenv("APP_LEGO_WORKING_DIR"), "/")
	repositoriesDir := workDir + "/repositories"
	customFilesDir := workDir + "/custom_files"
	var s service.Container
	s.Repository = repository.NewPostgres(pgConn, pgSchema)
	s.Branches = branch.NewPostgres(pgConn, pgSchema)
	s.Deployment = deployment.NewPostgres(pgConn, pgSchema)
	s.OS = appOs.NewOS()
	s.Variable = variable.NewVariable(marshaller.NewYaml(), s.Repository, customFilesDir)
	s.Validation = validation.NewValidation()
	s.VCS = vcs.NewGit(repositoriesDir, s.OS, s.Variable, marshaller.NewYaml())
	s.Builder = builder.NewBuilder(repositoriesDir, s.VCS, s.OS, s.Repository, s.Branches, s.Variable, marshaller.NewYaml())
	s.Deployer = deployer.NewDeployer(s.Repository, s.Branches, s.Deployment, s.OS, s.Variable, marshaller.NewYaml(), workDir)
	c := controller.NewController(s)
	go c.DownloadRepositoryJob(ctx)
	go c.SyncRepositoryJob(ctx)
	go c.BuildBranchJob(ctx)
	go c.WatchDeploymentsJob(ctx)
	runHttpServer(c)
}

func postgresConn(ctx context.Context) (*pgxpool.Pool, error) {
	pgs := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("APP_LEGO_DB_HOST"),
		os.Getenv("APP_LEGO_DB_PORT"),
		os.Getenv("APP_LEGO_DB_USER"),
		os.Getenv("APP_LEGO_DB_PASSWORD"),
		os.Getenv("APP_LEGO_DB_NAME"),
	)
	return pgxpool.Connect(ctx, pgs)
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
