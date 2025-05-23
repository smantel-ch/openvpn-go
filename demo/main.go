package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/smantel-ch/openvpn-go"
	"go.uber.org/zap"
)

func main() {
	username := flag.String("user", "", "VPN username")
	password := flag.String("pass", "", "VPN password")
	configPath := flag.String("config", "", "Path to .ovpn config file")
	debug := flag.Bool("debug", false, "Enable debug logging")
	timeout := flag.Int("timeout", 15, "Timeout in seconds for connection")

	flag.Parse()

	if *username == "" || *password == "" || *configPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *debug {
		logger, _ := zap.NewDevelopment()
		openvpn.SetLogger(logger.Sugar())
	}

	configData, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	client, err := openvpn.NewVPNClient(configData, *username, *password)
	if err != nil {
		log.Fatalf("Failed to initialize VPN client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	fmt.Println("Connecting to VPN...")
	if err := client.Connect(ctx); err != nil {
		if errors.Is(err, openvpn.ErrConnectionFailed) {
			log.Printf("VPN connection failed: %v", err)
		} else if errors.Is(err, openvpn.ErrTimeout) {
			log.Printf("VPN connection timed out.")
		} else {
			log.Fatalf("Unexpected VPN error: %v", err)
		}
	}

	go func() {
		for logLine := range client.LogsChan() {
			fmt.Println("[VPN LOG]", logLine)
		}
	}()

	fmt.Printf("Connected! Status: %s\n", client.Status())
	fmt.Println("Sleeping for 5 seconds...")
	time.Sleep(5 * time.Second)

	fmt.Println("Disconnecting...")
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client.DisconnectAndWait(waitCtx)
	fmt.Printf("Final Status: %s\n", client.Status())
}
