package main

import (
	"encoding/json"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

type TgBot struct {
	cfg                 TgBotCfg
	botApi              *tgbotapi.BotAPI
	updatesChan         tgbotapi.UpdatesChannel
	amqpPubConn         *amqp.Connection
	amqpConsumerConn    *amqp.Connection
	amqpPubChannel      *amqp.Channel
	amqpConsumerChannel *amqp.Channel
	amqpDone            chan bool
}

type TgBotCfg struct {
	BotToken    string
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

func NewTgBot(cfg TgBotCfg) *TgBot {
	botApi, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Panic(err)
	}

	botApi.Debug = true

	log.Printf("Authorized on account %s", botApi.Self.UserName)

	amqpPubConn, err := amqp.Dial(cfg.AmqpDSN)
	if err != nil {
		log.Panic(err)
	}
	amqpConsumerConn, err := amqp.Dial(cfg.AmqpDSN)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Connected to %s", cfg.AmqpDSN)

	bot := TgBot{
		cfg:              cfg,
		botApi:           botApi,
		amqpPubConn:      amqpPubConn,
		amqpConsumerConn: amqpConsumerConn,
	}

	return &bot
}

func (b *TgBot) Start() error {
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
	go b.handleAmqp(deliveries)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	b.updatesChan = b.botApi.GetUpdatesChan(u)
	go func() {
		for update := range b.updatesChan {
			if update.Message == nil {
				continue
			}
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := BotMessage{
				ChatId:  update.FromChat().ID,
				Text:    update.Message.Text,
				Command: update.Message.Command(),
			}
			b.sendToQueue(msg)
		}
	}()
	return nil
}

func (b *TgBot) handleAmqp(deliveries <-chan amqp.Delivery) {
	for d := range deliveries {
		//do_work
		var msg BotMessage
		err := json.Unmarshal(d.Body, &msg)
		if err != nil {
			log.Printf("unmarshal failed: %s\n", err)
			continue
		}

		err = b.sendToTelegram(msg)
		if err != nil {
			log.Printf("sendToTelegram failed: %s\n", err)
			continue
		}

		log.Printf("message sent: %s\n", msg.Text)
		d.Ack(false)
	}
	close(b.amqpDone)
}

func (b *TgBot) sendToTelegram(msg BotMessage) error {
	tgMsg := tgbotapi.NewMessage(msg.ChatId, msg.Text)
	resp, err := b.botApi.Send(tgMsg)
	if err != nil {
		return err
	}
	log.Printf("Tg API resp: %+v\n", resp)
	return nil
}

func (b *TgBot) Stop() error {
	b.botApi.StopReceivingUpdates()
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

func (b *TgBot) sendToQueue(msg BotMessage) error {
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
