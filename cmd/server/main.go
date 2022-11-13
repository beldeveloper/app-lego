package main

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/internal/app"
	"github.com/beldeveloper/app-lego/internal/app/svc"
	"github.com/beldeveloper/app-lego/rpc/hook"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/julienschmidt/httprouter"
	"google.golang.org/grpc"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// get watcher and router using DI wire
	c, err := initializeContainer()
	if err != nil {
		log.Fatalf("main: %v\n", err)
	}
	// run watcher that maintains the GIT repositories in background
	go c.watcher.Watch()
	// run http server
	runHttpServer(c.router)
}

type container struct {
	watcher svc.Watcher
	router  *httprouter.Router
}

func newContainer(watcher svc.Watcher, router *httprouter.Router) container {
	return container{
		watcher: watcher,
		router:  router,
	}
}

func reposDir() app.ReposDir {
	return app.ReposDir(os.Getenv("APP_LEGO_REPOS_DIR"))
}

func newAccessKey() app.ApiAccessKey {
	return app.ApiAccessKey(os.Getenv("APP_LEGO_ACCESS_KEY"))
}

func newWatcher(repo app.RepositorySvc, branch app.BranchSvc, deploy app.DeploymentSvc) svc.Watcher {
	return svc.NewWatcher([]app.WatcherJob{
		{
			Name: "downloadRepo",
			Do:   repo.DownloadJob,
		},
		{
			Name: "syncRepo",
			Do:   repo.SyncJob,
		},
		{
			Name: "buildBranch",
			Do:   branch.BuildJob,
		},
		{
			Name: "watchDeploy",
			Do:   deploy.WatchJob,
		},
	})
}

func newPostgresConn() *pgxpool.Pool {
	pgs := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("APP_LEGO_DB_HOST"),
		os.Getenv("APP_LEGO_DB_PORT"),
		os.Getenv("APP_LEGO_DB_USER"),
		os.Getenv("APP_LEGO_DB_PASSWORD"),
		os.Getenv("APP_LEGO_DB_NAME"),
	)
	conn, err := pgxpool.Connect(context.Background(), pgs)
	if err != nil {
		log.Fatalf("main.newPostgresConn: %v\n", err)
	}
	return conn
}

func newHookConn() hook.HookClient {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	addr := os.Getenv("APP_LEGO_HOOK_HANDLER_ADDR")
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("main.newHookConn: dial: %v; addr=%s\n", err, addr)
	}
	return hook.NewHookClient(conn)
}

func runHttpServer(router *httprouter.Router) {
	httpPort := os.Getenv("APP_LEGO_HTTP_PORT")
	crtFile := os.Getenv("APP_LEGO_HTTPS_CRT")
	keyFile := os.Getenv("APP_LEGO_HTTPS_KEY")
	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: router,
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		var err error
		if len(crtFile) > 0 {
			err = srv.ListenAndServeTLS(crtFile, keyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("main.runHttpServer: serve http: %v; port = %s\n", err, httpPort)
		}
	}()
	log.Printf("Listening :%s for HTTP connections...\n", httpPort)
	<-done
	log.Print("Stopping the application...\n")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("main.runHttpServer: server shutdown: %v\n", err)
	}
}
