package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mobentum/kern"
	"github.com/mobentum/kern/extensions/config"
)

type Config struct {
	Host    string
	Port    int
	Debug   bool
	Timeout time.Duration
}

func LoadConfig() (*Config, error) {
	loader, err := config.New(
		config.WithPrefix("APP"),
		config.WithDotEnv(".env"),
	)
	if err != nil {
		return nil, err
	}

	port, err := loader.Int("PORT", 8080)
	if err != nil {
		return nil, err
	}
	debug, err := loader.Bool("DEBUG", true)
	if err != nil {
		return nil, err
	}
	timeout, err := loader.Duration("TIMEOUT", 5*time.Second)
	if err != nil {
		return nil, err
	}

	return &Config{
		Host:    loader.String("HOST", "127.0.0.1"),
		Port:    port,
		Debug:   debug,
		Timeout: timeout,
	}, nil
}

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	app := kern.Default()
	app.GET("/config", func(c *kern.Context) {
		_ = c.JSON(http.StatusOK, map[string]interface{}{
			"host":    cfg.Host,
			"port":    cfg.Port,
			"debug":   cfg.Debug,
			"timeout": cfg.Timeout.String(),
		})
	})

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Fatal(app.Run(addr))
}
