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

	cfg := CoreConfig{
		AmqpDSN:     amqpDSN,
		SendQ:       "queue1",
		RecvQ:       "queue2",
		ConsumerTag: "tgbot",
	}
	botCore := NewBotCore(cfg)

	tgBotCfg := TgBotCfg{
		BotToken: botToken,
		Core:     botCore,
	}
	tgBot := NewTgBot(tgBotCfg)

	botCore.AddHandler(BotTypeTelegram, func(coreMsg CoreMessage) error {
		err := tgBot.Send(coreMsg)
		if err != nil {
			return fmt.Errorf("telegram Send failed: %w", err)
		}
		return nil
	})

	err := botCore.Start()
	if err != nil {
		log.Panic(err)
	}
	tgBot.Start()
	<-exit

	fmt.Println("Stopping...")
	tgBot.Stop()
	err = botCore.Stop()
	if err != nil {
		log.Panic(err)
	}
}
