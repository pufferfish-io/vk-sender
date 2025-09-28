package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	cfg "vk-sender/internal/config"
	"vk-sender/internal/logger"
	"vk-sender/internal/messaging"
	"vk-sender/internal/processor"
)

func main() {
	lg, cleanup := logger.NewZapLogger()
	defer cleanup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lg.Info("🚀 Starting vk-sender…")

	conf, err := cfg.Load()
	if err != nil {
		lg.Error("❌ Failed to load config: %v", err)
		os.Exit(1)
	}

	sender := processor.NewVkMessageSender(processor.Option{
		Token:  conf.VK.Token,
		Logger: lg,
	})

	consumer, err := messaging.NewKafkaConsumer(messaging.ConsumerOption{
		Logger:       lg,
		Broker:       conf.Kafka.BootstrapServersValue,
		GroupID:      conf.Kafka.ResponseMessageGroupID,
		Topics:       []string{conf.Kafka.VkMessageTopicName},
		Handler:      sender,
		SaslUsername: conf.Kafka.SaslUsername,
		SaslPassword: conf.Kafka.SaslPassword,
		ClientID:     conf.Kafka.ClientID,
	})
	if err != nil {
		lg.Error("❌ Failed to create consumer: %v", err)
		os.Exit(1)
	}

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		cancel()
	}()

	if err := consumer.Start(ctx); err != nil {
		lg.Error("❌ Consumer error: %v", err)
		os.Exit(1)
	}
}
