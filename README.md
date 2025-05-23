# OpenVPN Client (Go Module)
A lightweight, extensible Go module that wraps the OpenVPN CLI — designed for API and CLI-based VPN management. It provides secure connection handling, automatic cleanup, and contextual lifecycle control.

---

## 🚀 Features
- ✅ Start/Stop/Reconnect OpenVPN securely
- 🔐 Internal handling of username & password (never exposed)
- 🧹 Temporary file cleanup after use
- 📡 Live log & status streaming
- ⛔ Custom error types (`ErrTimeout`, `ErrAlreadyRunning`, etc.)
- ⚙️ Built-in tests and automation via `Makefile`

---

## 📦 Installation
```bash
go  get  github.com/smantel-ch/openvpn-go
```


## ✨ Example Usage
```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/smantel-ch/openvpn-go"
)

func main() {
	config := []byte("...your .ovpn content...")

	client, err := openvpn.NewVPNClient(config, "myuser", "mypass")
	if err != nil {
		log.Fatal("init error:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		log.Fatal("connection failed:", err)
	}

	fmt.Println("VPN Status:", client.Status())
	client.Disconnect()
}
```


## 🧪 Testing
Run the test suite:
```bash
make test
```

Run tests with coverage:
```bash
make test-cover
```


## 🔧 Dev Commands
```bash
make # runs fmt, lint, test
make fmt # gofmt formatting
make lint # golangci-lint
make build # builds CLI (cmd/demo)
make ci # full local pipeline check
```


## 🖥️ Demo CLI
```bash
go run ./demo/main.go -user myuser -pass mypass -config my.ovpn
```
