package messaging

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"vk-sender/internal/logger"

	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"
)

type Handler interface {
	Handle(ctx context.Context, raw []byte) error
}

type KafkaConsumer struct {
	group   sarama.ConsumerGroup
	handler Handler
	topics  []string
	log     logger.Logger
	cancel  context.CancelFunc
}

type ConsumerOption struct {
	Logger       logger.Logger
	Broker       string
	GroupID      string
	Topics       []string
	Handler      Handler
	SaslUsername string
	SaslPassword string
	ClientID     string
}

func NewKafkaConsumer(opt ConsumerOption) (*KafkaConsumer, error) {
	cfg := newSaramaConfig(opt)

	group, err := sarama.NewConsumerGroup([]string{opt.Broker}, opt.GroupID, cfg)
	if err != nil {
		return nil, fmt.Errorf("kafka consumer init: %w", err)
	}
	opt.Logger.Info("Kafka Consumer init: broker=%s group=%s topics=%s", opt.Broker, opt.GroupID, strings.Join(opt.Topics, ","))
	return &KafkaConsumer{group: group, handler: opt.Handler, topics: opt.Topics, log: opt.Logger}, nil
}

func (c *KafkaConsumer) Start(ctx context.Context) error {
	if c.group == nil {
		return errors.New("kafka consumer group is nil")
	}

	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	h := &consumerGroupHandler{handle: c.handler, log: c.log}

	defer c.group.Close()

	go c.watchErrors(ctx)

	c.log.Info("consumer starting: topics=%s", strings.Join(c.topics, ","))

	for ctx.Err() == nil {
		if err := c.group.Consume(ctx, c.topics, h); err != nil {
			return fmt.Errorf("consume: %w", err)
		}
	}

	c.cancel = nil
	return nil
}

func (c *KafkaConsumer) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

type consumerGroupHandler struct {
	handle Handler
	log    logger.Logger
}

func (h *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.log.Info("consumer group claims: %v", session.Claims())
	return nil
}

func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("handler panic: %v", r)
				}
			}()
			err = h.handle.Handle(session.Context(), msg.Value)
		}()
		if err == nil {
			session.MarkMessage(msg, "")
			h.log.Debug("message handled: topic=%s partition=%d offset=%d bytes=%d", msg.Topic, msg.Partition, msg.Offset, len(msg.Value))
		} else {
			h.log.Error("handler error: topic=%s partition=%d offset=%d err=%v", msg.Topic, msg.Partition, msg.Offset, err)
		}
	}
	return nil
}

func newSaramaConfig(opt ConsumerOption) *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.ClientID = opt.ClientID

	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	cfg.Consumer.Group.Session.Timeout = 15 * time.Second
	cfg.Consumer.Group.Heartbeat.Interval = 3 * time.Second

	cfg.Net.SASL.Enable = true
	cfg.Net.SASL.User = opt.SaslUsername
	cfg.Net.SASL.Password = opt.SaslPassword
	cfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
	cfg.Net.SASL.Handshake = true
	cfg.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
		return &xdgSCRAMClient{hash: scram.SHA512}
	}
	cfg.Version = sarama.V2_8_0_0
	return cfg
}

func (c *KafkaConsumer) watchErrors(ctx context.Context) {
	if c.group == nil {
		return
	}
	for {
		select {
		case err, ok := <-c.group.Errors():
			if !ok {
				return
			}
			c.log.Error("consumer group error: %v", err)
		case <-ctx.Done():
			return
		}
	}
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
