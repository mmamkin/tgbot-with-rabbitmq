package main

import (
	"fmt"
	"log"
	"strings"
)

// Finite state machine

const (
	msgWhichPizza       = "Какую пиццу вы хотите? Большую или маленькую?"
	msgWhichPaymentType = "Как вы будете платить?"
	msgApproveTpl       = "Вы хотите %s пиццу, оплата %s?"
	msgThankYou         = "Спасибо за заказ"
)

type Event struct {
	Message string
	Command string
}

const (
	stWaitForStart = iota
	stWaitForSize
	stWaitPaymentType
	stWaitForApprove
	stFinish
)

type Fsm struct {
	BotType     string
	ChatId      string
	State       int
	PizzaSize   string
	PaymentType string
}

type Action struct {
	Message string
}

func NewFsm(botType string, chatId string) *Fsm {
	return &Fsm{
		BotType: botType,
		ChatId:  chatId,
		State:   stWaitForStart,
	}
}

func (f *Fsm) StateName(state int) string {
	switch state {
	case stWaitForStart:
		return "WaitForStart"
	case stWaitForSize:
		return "WaitForSize"
	case stWaitPaymentType:
		return "WaitPaymentType"
	case stWaitForApprove:
		return "WaitForApprove"
	case stFinish:
		return "Finish"
	}
	return "unknown state"
}

// Step реализует шаг автомата и возвращает слайс действий (сообщений для отправки)
func (f *Fsm) Step(e Event) []Action {
	var actions []Action

	oldState := f.State
	switch f.State {
	case stWaitForStart:
		if e.Command == "start" {
			f.State = stWaitForSize
			actions = append(actions, Action{Message: msgWhichPizza})
		}
	case stWaitForSize:
		msgLower := strings.ToLower(e.Message)
		if msgLower == "большую" || msgLower == "маленькую" {
			f.State = stWaitPaymentType
			f.PizzaSize = msgLower
			actions = append(actions, Action{Message: msgWhichPaymentType})
		} else {
			// todo: отправить пользователю сообщение об ошибке?
		}
	case stWaitPaymentType:
		msgLower := strings.ToLower(e.Message)
		if msgLower == "наличкой" || msgLower == "картой" {
			f.State = stWaitForApprove
			f.PaymentType = msgLower
			msgAskForApprove := fmt.Sprintf(msgApproveTpl, f.PizzaSize, f.PaymentType)
			actions = append(actions, Action{Message: msgAskForApprove})
		} else {
			// todo: отправить пользователю сообщение об ошибке?
		}
	case stWaitForApprove:
		msgLower := strings.ToLower(e.Message)
		if msgLower == "да" {
			// f.State = stFinish
			// для простоты возвращаемся к начальному состоянию
			f.State = stWaitForStart
			actions = append(actions, Action{Message: msgThankYou})
		} else {
			// todo: отправить пользователю сообщение об ошибке?
		}
	}

	if f.State != oldState {
		log.Printf(
			"[DEBUG] botType %s, chatId %s, state from %s to %s",
			f.BotType,
			f.ChatId,
			f.StateName(oldState),
			f.StateName(f.State),
		)
	}
	return actions
}
