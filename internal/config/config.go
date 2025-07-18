package config

import (
	"errors"
	"os"
	"sync"
)

type Config struct {
	Env       string
	LogLevel  string
	DBType    string
	DBDSN     string
	FileSleep string
	FileGoals string
}

var (
	cfg  *Config
	once sync.Once
)

func Load() *Config {
	once.Do(func() {
		_ = loadDotEnv()
		cfg = &Config{
			Env:       getEnv("APP_ENV", "development"),
			LogLevel:  getEnv("LOG_LEVEL", "info"),
			DBType:    getEnv("STORAGE_BACKEND", "file"),
			DBDSN:     getEnv("POSTGRES_DSN", ""),
			FileSleep: getEnv("SLEEP_FILE", "data/sleep_logs.json"),
			FileGoals: getEnv("GOALS_FILE", "data/goals.json"),
		}
		if err := cfg.Validate(); err != nil {
			panic("Invalid config: " + err.Error())
		}
	})
	return cfg
}

func (c *Config) Validate() error {
	if c.DBType == "postgres" && c.DBDSN == "" {
		return errors.New("POSTGRES_DSN is required when STORAGE_BACKEND=postgres")
	}
	if c.DBType == "file" && (c.FileSleep == "" || c.FileGoals == "") {
		return errors.New("File storage requires SLEEP_FILE and GOALS_FILE to be set")
	}
	if c.Env != "development" && c.Env != "staging" && c.Env != "production" {
		return errors.New("APP_ENV must be one of: development, staging, production")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadDotEnv() error {
	if _, err := os.Stat(".env"); err == nil {
		f, err := os.Open(".env")
		if err != nil {
			return err
		}
		defer f.Close()
		var lines []string
		buf := make([]byte, 4096)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				lines = append(lines, string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
		for _, line := range lines {
			for _, l := range splitLines(line) {
				if len(l) == 0 || l[0] == '#' {
					continue
				}
				kv := splitKV(l)
				if len(kv) == 2 {
					os.Setenv(kv[0], kv[1])
				}
			}
		}
	}
	return nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' || c == '\r' {
			if i > start {
				lines = append(lines, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitKV(s string) []string {
	for i, c := range s {
		if c == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
