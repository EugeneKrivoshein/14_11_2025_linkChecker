package main

import (
	"14_11_2025_linkChecker/internal/handlers"
	"14_11_2025_linkChecker/internal/routes"
	"14_11_2025_linkChecker/internal/store"
	"14_11_2025_linkChecker/internal/worker"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	st, err := store.NewFileStore("./data")
	if err != nil {
		log.Fatal(err)
	}

	mgr := worker.NewManager(st, 5)
	go mgr.Run()

	h := handlers.NewHandler(st, mgr)
	router := routes.NewRouter(h)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Server started on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}

	mgr.Stop()

	log.Println("Server exited gracefully")
}
