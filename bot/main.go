package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	botToken := os.Getenv("BOT_TOKEN")
	amqpDSN := os.Getenv("AMQP_DSN")

	cfg := TgBotCfg{
		BotToken:    botToken,
		AmqpDSN:     amqpDSN,
		SendQ:       "queue1",
		RecvQ:       "queue2",
		ConsumerTag: "tgbot",
	}
	bot := NewTgBot(cfg)
	err := bot.Start()
	if err != nil {
		log.Panic(err)
	}
	<-exit
	fmt.Println("Stopping...")
	err = bot.Stop()
	if err != nil {
		log.Panic(err)
	}
}
