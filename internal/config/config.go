package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port  string      `yaml:"port"`
	MySQL MySQLConfig `yaml:"mysql"`
	JWT   JWTConfig   `yaml:"jwt"`
	COS   COSConfig   `yaml:"cos"`
}

type MySQLConfig struct {
	DSN string
}

type JWTConfig struct {
	Secret        string `yaml:"secret"`
	ExpireMinutes int    `yaml:"expire_minutes"`
}

type COSConfig struct {
	SecretID  string `yaml:"secret_id"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	Region    string `yaml:"region"`
}

func LoadConfig(configFile string) *Config {
	cfg := &Config{}
	data, err := os.ReadFile(configFile)
	if err != nil {
		return cfg
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Fatalf("failed to parse config.yaml: %v", err)
	}
	return cfg
}
