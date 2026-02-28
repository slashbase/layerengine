package config

import (
	"os"
)

type Config struct {
	Version   string
	IsLive    bool
	OpenAIKey string
}

var conf *Config

func Init(version string) {
	if conf == nil {
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
