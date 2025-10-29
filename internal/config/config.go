package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

type Kafka struct {
	BootstrapServersValue  string `validate:"required" env:"BOOTSTRAP_SERVERS_VALUE"`
	VkMessageTopicName     string `validate:"required" env:"VK_MESSAGE_TOPIC_NAME"`
	ResponseMessageGroupID string `validate:"required" env:"RESPONSE_MESSAGE_GROUP_ID"`
	SaslUsername           string `env:"SASL_USERNAME"`
	SaslPassword           string `env:"SASL_PASSWORD"`
	ClientID               string `env:"CLIENT_ID"`
}

type Server struct {
	Port int `validate:"required" env:"PORT"`
}

type Vk struct {
	Token string `validate:"required" env:"TOKEN"`
}

type Config struct {
	Server Server `envPrefix:"VKSENDER_SERVER_"`
	Kafka  Kafka  `envPrefix:"KAFKA_"`
	VK     Vk     `envPrefix:"VK_"`
}

func Load() (*Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return nil, fmt.Errorf("env parse: %w", err)
	}
	v := validator.New()
	if err := v.Struct(c); err != nil {
		return nil, fmt.Errorf("config validate: %w", err)
	}
	return &c, nil
}
