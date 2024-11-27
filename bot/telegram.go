package main

import (
	"fmt"
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mmamkin/tgbot-with-rabbitmq/internal/core"
)

const BotTypeTelegram = "telegram"

type TgBotCfg struct {
	BotToken string
	Core     *BotCore
}

type TgBot struct {
	cfg         TgBotCfg
	botApi      *tgbotapi.BotAPI
	updatesChan tgbotapi.UpdatesChannel
	done        chan bool
}

func NewTgBot(cfg TgBotCfg) *TgBot {
	botApi, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Panic(err)
	}

	botApi.Debug = true

	log.Printf("Authorized on account %s", botApi.Self.UserName)

	bot := TgBot{
		cfg:    cfg,
		botApi: botApi,
	}

	return &bot
}

func (b *TgBot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	b.updatesChan = b.botApi.GetUpdatesChan(u)
	b.done = make(chan bool)
	go func() {
		for update := range b.updatesChan {
			if update.Message == nil {
				continue
			}
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			coreMsg := core.Message{
				BotType: BotTypeTelegram,
				ChatId:  strconv.FormatInt(update.FromChat().ID, 10),
				Text:    update.Message.Text,
				Command: update.Message.Command(),
			}
			b.cfg.Core.SendToQueue(coreMsg)
		}
		close(b.done)
	}()
}
func (b *TgBot) Stop() {
	b.botApi.StopReceivingUpdates()
	<-b.done
}

func (b *TgBot) Send(coreMsg core.Message) error {
	chatId, err := strconv.ParseInt(coreMsg.ChatId, 10, 64)
	if err != nil {
		return fmt.Errorf("bad ChatId: %s", coreMsg.ChatId)
	}
	tgMsg := tgbotapi.NewMessage(chatId, coreMsg.Text)
	resp, err := b.botApi.Send(tgMsg)
	if err != nil {
		return err
	}
	log.Printf("Tg API resp: %+v\n", resp)
	return nil
}
