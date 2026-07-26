package main

import (
	"context"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cs2-demo-service/api"
	"cs2-demo-service/db"
	"cs2-demo-service/middlewares"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	// Cargar el fichero .env desde la raíz del proyecto
	err := godotenv.Load("../.env")
	if err != nil {
		log.Println("No se pudo cargar el fichero .env (no es crítico):", err)
	}
	if err := db.InitRedis(); err != nil {
		log.Fatal(err)
	}
	defer db.Rdb.Close()

	// Crea el router.
	router := mux.NewRouter()

	// Endpoint simplificado: procesa una demo y devuelve el JSON directamente
	// Protected: only accessible from localhost (Node.js service)
	router.HandleFunc("/process-demo", middlewares.WithInternalOnly(api.HandleProcessDemo)).Methods("POST")
	router.HandleFunc("/health", api.HandleHealth).Methods("GET")
	router.HandleFunc("/ready", api.HandleReady).Methods("GET")

	handler := middlewares.WithSecurityHeaders(router)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 * 1024,
	}

	adminMux := http.NewServeMux()
	adminMux.Handle("/metrics", promhttp.Handler())
	if os.Getenv("ENABLE_PPROF") == "true" {
		adminMux.HandleFunc("/debug/pprof/", pprof.Index)
		adminMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		adminMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		adminMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		adminMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	adminAddr := os.Getenv("GO_ADMIN_ADDR")
	if adminAddr == "" {
		adminAddr = "127.0.0.1:6060"
	}
	adminServer := &http.Server{
		Addr:              adminAddr,
		Handler:           adminMux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("admin server failed: %v", err)
		}
	}()

	log.Println("🚀 Servicio de análisis de demos CS2 iniciado en puerto :8080")
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = adminServer.Shutdown(shutdownContext)
	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("HTTP shutdown failed: %v", err)
	}
}
