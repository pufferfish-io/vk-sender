package messaging

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"vk-sender/internal/logger"

	"github.com/IBM/sarama"
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
	Context      context.Context
}

func NewKafkaConsumer(opt ConsumerOption) (*KafkaConsumer, error) {
	cfg := newConsumerConfig(opt)
	ctx := ensureContext(opt.Context)

	var group sarama.ConsumerGroup
	if err := connectWithRetry(ctx, opt.Logger, "Kafka consumer", func() error {
		var err error
		group, err = sarama.NewConsumerGroup([]string{opt.Broker}, opt.GroupID, cfg)
		return err
	}); err != nil {
		return nil, err
	}

	if opt.Logger != nil {
		opt.Logger.Info("Kafka Consumer ready: broker=%s group=%s topics=%s", opt.Broker, opt.GroupID, strings.Join(opt.Topics, ","))
	}

	return &KafkaConsumer{
		group:   group,
		handler: opt.Handler,
		topics:  opt.Topics,
		log:     opt.Logger,
	}, nil
}

func (c *KafkaConsumer) Start(ctx context.Context) error {
	if c.group == nil {
		return errors.New("kafka consumer group is nil")
	}

	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	handler := &consumerGroupHandler{handle: c.handler, log: c.log}

	defer c.group.Close()

	go c.watchErrors(ctx)

	if c.log != nil {
		c.log.Info("consumer starting: topics=%s", strings.Join(c.topics, ","))
	}

	for ctx.Err() == nil {
		if err := c.group.Consume(ctx, c.topics, handler); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if c.log != nil {
				c.log.Error("consumer error: %v", err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(consumeRetryDelay):
			}
			continue
		}
	}

	c.cancel = nil
	return ctx.Err()
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
	if h.log != nil {
		h.log.Info("consumer group claims: %v", session.Claims())
	}
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
			if h.log != nil {
				h.log.Debug("message handled: topic=%s partition=%d offset=%d bytes=%d", msg.Topic, msg.Partition, msg.Offset, len(msg.Value))
			}
		} else if h.log != nil {
			h.log.Error("handler error: topic=%s partition=%d offset=%d err=%v", msg.Topic, msg.Partition, msg.Offset, err)
		}
	}
	return nil
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
			if c.log != nil {
				c.log.Error("consumer group error: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
