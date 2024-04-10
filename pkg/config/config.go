package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Version   string
	IsLive    bool
	OpenAIKey string
}

var conf *Config

func Init(version string) {
	if conf == nil {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
		conf = &Config{
			Version:   version,
			IsLive:    false,
			OpenAIKey: os.Getenv("OPENAI_SECRET_KEY"),
		}
	}
}

func Get() *Config {
	return conf
}
