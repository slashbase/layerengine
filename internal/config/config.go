package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	IsLive    bool
	OpenAIKey string
}

var conf *Config

func Get() *Config {
	if conf == nil {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
		conf = &Config{
			IsLive:    false,
			OpenAIKey: os.Getenv("OPENAI_SECRET_KEY"),
		}
	}
	return conf
}
