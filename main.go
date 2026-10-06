package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"socks5-proxy/server"
)

func main() {
	host := flag.String("host", "0.0.0.0", "Listen address")
	port := flag.Int("port", 1080, "Listen port")
	user := flag.String("user", "", "Username for authentication (optional)")
	pass := flag.String("pass", "", "Password for authentication (requires -user)")
	logFile := flag.String("log", "", "Log file path (default: stderr)")
	flag.Parse()

	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			log.Fatalf("Cannot open log file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	cfg := server.Config{
		Host: *host,
		Port: *port,
	}

	if *user != "" {
		if *pass == "" {
			fmt.Fprintln(os.Stderr, "Error: -pass is required when -user is set")
			os.Exit(1)
		}
		cfg.Username = *user
		cfg.Password = *pass
	}

	srv := server.New(cfg)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		srv.Close()
		os.Exit(0)
	}()

	log.Printf("SOCKS5 proxy listening on %s:%d", cfg.Host, cfg.Port)
	if cfg.Username != "" {
		log.Println("Authentication: enabled")
	} else {
		log.Println("Authentication: disabled (open proxy)")
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
