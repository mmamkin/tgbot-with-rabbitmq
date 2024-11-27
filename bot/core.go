package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/mmamkin/tgbot-with-rabbitmq/internal/core"
	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageHandlerFunc func(msg core.Message) error

type BotCore struct {
	cfg                 core.Config
	amqpPubConn         *amqp.Connection
	amqpConsumerConn    *amqp.Connection
	amqpPubChannel      *amqp.Channel
	amqpConsumerChannel *amqp.Channel
	amqpDone            chan bool
	messageHandlers     map[string]MessageHandlerFunc
}

func NewBotCore(cfg core.Config) *BotCore {
	amqpPubConn, err := amqp.Dial(cfg.AmqpDSN)
	if err != nil {
		log.Panic(err)
	}
	amqpConsumerConn, err := amqp.Dial(cfg.AmqpDSN)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Connected to %s", cfg.AmqpDSN)

	return &BotCore{
		cfg:              cfg,
		amqpPubConn:      amqpPubConn,
		amqpConsumerConn: amqpConsumerConn,
		messageHandlers:  make(map[string]MessageHandlerFunc),
	}
}

func (b *BotCore) Start() error {
	var err error
	b.amqpConsumerChannel, err = b.amqpConsumerConn.Channel()
	if err != nil {
		return fmt.Errorf("Channel() failed: %w", err)
	}

	b.amqpPubChannel, err = b.amqpPubConn.Channel()
	if err != nil {
		return fmt.Errorf("Channel() failed: %w", err)
	}

	deliveries, err := b.amqpConsumerChannel.Consume(
		b.cfg.RecvQ,       // name
		b.cfg.ConsumerTag, // consumerTag,
		false,             // autoAck
		false,             // exclusive
		false,             // noLocal
		false,             // noWait
		nil,               // arguments
	)
	if err != nil {
		return fmt.Errorf("amqp Consume() failed: %w", err)
	}
	b.amqpDone = make(chan bool)
	go b.handleAmqp(deliveries)
	return nil
}

func (b *BotCore) AddHandler(botType string, handler MessageHandlerFunc) {
	b.messageHandlers[botType] = handler
}

func (b *BotCore) handleAmqp(deliveries <-chan amqp.Delivery) {
	for d := range deliveries {
		if err := b.handleDelivery(&d); err != nil {
			d.Nack(false, false)
		} else {
			d.Ack(false)
		}
	}
	close(b.amqpDone)
}

func (b *BotCore) handleDelivery(d *amqp.Delivery) error {
	var msg core.Message
	err := json.Unmarshal(d.Body, &msg)
	if err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	handler := b.messageHandlers[msg.BotType]
	if handler == nil {
		return fmt.Errorf("handler not found for botType '%s'", msg.BotType)
	}
	err = handler(msg)
	if err != nil {
		return fmt.Errorf("handler failed: %w", err)
	}
	return nil
}

func (b *BotCore) Stop() error {
	err := b.amqpConsumerChannel.Cancel(b.cfg.ConsumerTag, false)
	if err != nil {
		return err
	}
	<-b.amqpDone

	err = b.amqpConsumerConn.Close()
	if err != nil {
		return err
	}

	err = b.amqpPubConn.Close()
	if err != nil {
		return err
	}
	return nil
}

func (b *BotCore) SendToQueue(msg core.Message) error {
	bytes, err := json.Marshal(&msg)
	if err != nil {
		return err
	}

	amqpMsg := amqp.Publishing{
		Body: bytes,
	}

	log.Printf("Publishing to queue '%s'", b.cfg.SendQ)

	err = b.amqpPubChannel.Publish("", b.cfg.SendQ, false, false, amqpMsg)
	if err != nil {
		return err
	}

	return nil
}
