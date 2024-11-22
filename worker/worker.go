package main

import (
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Worker struct {
	cfg                 WorkerCfg
	amqpPubConn         *amqp.Connection
	amqpConsumerConn    *amqp.Connection
	amqpPubChannel      *amqp.Channel
	amqpConsumerChannel *amqp.Channel
	amqpDone            chan bool
	chatStates          map[int64]*Fsm
}

type WorkerCfg struct {
	SendQ       string
	RecvQ       string
	AmqpDSN     string
	ConsumerTag string
}

type BotMessage struct {
	ChatId  int64
	Text    string
	Command string
}

func NewWorker(cfg WorkerCfg) *Worker {

	amqpPubConn, err := amqp.Dial(cfg.AmqpDSN)
	if err != nil {
		log.Panic(err)
	}
	amqpConsumerConn, err := amqp.Dial(cfg.AmqpDSN)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Connected to %s", cfg.AmqpDSN)

	worker := Worker{
		cfg:              cfg,
		amqpPubConn:      amqpPubConn,
		amqpConsumerConn: amqpConsumerConn,
		chatStates:       make(map[int64]*Fsm),
	}

	return &worker
}

func (b *Worker) Start() error {
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
		return fmt.Errorf("queue Consume: %w", err)
	}
	b.amqpDone = make(chan bool)
	go b.handle(deliveries)
	return nil
}

func (b *Worker) handle(deliveries <-chan amqp.Delivery) {
	for d := range deliveries {
		var msg BotMessage
		json.Unmarshal(d.Body, &msg)
		chatId := msg.ChatId
		fsm, found := b.chatStates[chatId]
		if !found {
			fsm = NewFsm(chatId)
			b.chatStates[chatId] = fsm
			log.Printf("[DEBUG] fsm created for chatId %d\n", chatId)
		} else {
			log.Printf(
				"[DEBUG] fsm found for chatId %d in state %s\n",
				chatId,
				fsm.StateName(fsm.State),
			)
		}
		actions := fsm.Step(Event{Command: msg.Command, Message: msg.Text})
		for _, action := range actions {
			msgReply := BotMessage{
				ChatId: msg.ChatId,
				Text:   action.Message,
			}
			b.sendToQueue(msgReply)
		}

		log.Printf("message received: %s\n", string(d.Body))
		d.Ack(false)
	}
	close(b.amqpDone)
}

func (b *Worker) Stop() error {
	err := b.amqpConsumerChannel.Cancel(b.cfg.ConsumerTag, false)
	if err != nil {
		return err
	}

	<-b.amqpDone // wait for handling all deliveries

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

func (b *Worker) sendToQueue(msg BotMessage) error {
	bytes, err := json.Marshal(&msg)
	if err != nil {
		return err
	}

	amqpMsg := amqp.Publishing{
		Body: bytes,
	}

	log.Printf("Publishing to queue '%s'", b.cfg.SendQ)

	err = b.amqpPubChannel.Publish("", b.cfg.SendQ, false, false, amqpMsg)
	// todo: recreate channel on error?
	if err != nil {
		return err
	}

	return nil
}
