package main

import (
	"os"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}
