package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config defines the runtime configuration for the QuantSage server.
type Config struct {
	App struct {
		Name string `yaml:"name"`
		Env  string `yaml:"env"`
		Addr string `yaml:"addr"`
	} `yaml:"app"`
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
}

// Load reads YAML config and applies supported environment overrides.
func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	defer file.Close()

	var cfg Config
	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid config line: %q", line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		switch section {
		case "app":
			switch key {
			case "name":
				cfg.App.Name = value
			case "env":
				cfg.App.Env = value
			case "addr":
				cfg.App.Addr = value
			}
		case "database":
			if key == "dsn" {
				cfg.Database.DSN = value
			}
		case "redis":
			switch key {
			case "addr":
				cfg.Redis.Addr = value
			case "password":
				cfg.Redis.Password = value
			case "db":
				db, convErr := strconv.Atoi(value)
				if convErr != nil {
					return nil, fmt.Errorf("parse redis db: %w", convErr)
				}
				cfg.Redis.DB = db
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan config file: %w", err)
	}

	if v := os.Getenv("QUANTSAGE_DATABASE_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("QUANTSAGE_REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("QUANTSAGE_REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}

	return &cfg, nil
}
