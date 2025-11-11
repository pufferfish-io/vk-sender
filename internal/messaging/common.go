package messaging

import (
	"context"
	"errors"
	"time"

	"vk-sender/internal/logger"

	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"
)

const (
	retryInitialInterval = 500 * time.Millisecond
	retryMaxInterval     = 30 * time.Second
	consumeRetryDelay    = 2 * time.Second
)

var ErrKafkaUnavailable = errors.New("kafka unavailable")

func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func connectWithRetry(ctx context.Context, logger logger.Logger, label string, dial func() error) error {
	ctx = ensureContext(ctx)
	interval := retryInitialInterval

	for {
		if err := dial(); err == nil {
			return nil
		} else if logger != nil {
			logger.Error("%s connect failed: %v", label, err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}

		interval = shrinkBackoff(interval)
	}
}

func shrinkBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > retryMaxInterval {
		next = retryMaxInterval
	}
	return next
}

func newConsumerConfig(opt ConsumerOption) *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.ClientID = opt.ClientID
	cfg.Version = sarama.V2_8_0_0

	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	cfg.Consumer.Group.Session.Timeout = 15 * time.Second
	cfg.Consumer.Group.Heartbeat.Interval = 3 * time.Second
	cfg.Consumer.Retry.Backoff = 2 * time.Second

	cfg.Metadata.Retry.Max = 5
	cfg.Metadata.Retry.Backoff = 2 * time.Second

	cfg.Net.DialTimeout = 5 * time.Second
	cfg.Net.ReadTimeout = 5 * time.Second
	cfg.Net.WriteTimeout = 5 * time.Second

	configureSASL(cfg, opt.SaslUsername, opt.SaslPassword)

	return cfg
}

func configureSASL(cfg *sarama.Config, user, password string) {
	cfg.Net.SASL.Enable = true
	cfg.Net.SASL.User = user
	cfg.Net.SASL.Password = password
	cfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
	cfg.Net.SASL.Handshake = true
	cfg.Net.SASL.SCRAMClientGeneratorFunc = newScramClient
}

func newScramClient() sarama.SCRAMClient {
	return &xdgSCRAMClient{hash: scram.SHA512}
}

type xdgSCRAMClient struct {
	hash scram.HashGeneratorFcn
	*scram.Client
	*scram.ClientConversation
}

func (x *xdgSCRAMClient) Begin(userName, password, authzID string) error {
	c, err := x.hash.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	x.Client = c
	x.ClientConversation = c.NewConversation()
	return nil
}

func (x *xdgSCRAMClient) Step(challenge string) (string, error) {
	if x.ClientConversation == nil {
		return "", errors.New("no scram conversation")
	}
	return x.ClientConversation.Step(challenge)
}

func (x *xdgSCRAMClient) Done() bool {
	if x.ClientConversation == nil {
		return false
	}
	return x.ClientConversation.Done()
}
