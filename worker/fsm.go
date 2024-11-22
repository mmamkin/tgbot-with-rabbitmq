package main

import (
	"log"
	"strings"
)

// Finite state machine

const (
	stInit = iota
	stWaitForStart
	stWaitForSize
	stWaitPaymentType
	stWaitApprove
	stFinish
)

type Event struct {
	Message string
	Command string
}

type Fsm struct {
	ChatId int64
	State  int
}

func (f *Fsm) Step(e Event) {
	oldState := f.State
	switch f.State {
	case stInit:
		f.State = stWaitForStart
	case stWaitForStart:
		if e.Command == "start" {
			f.State = stWaitForSize
			f.SendMessage()
		}
	case stWaitForSize:
		msgLower := strings.ToLower(e.Message)
		if msgLower == "большую" {
			f.State = stWaitApprove
		} else if msgLower == "маленькую" {
			f.State = stWaitApprove
		}
	}

	if f.State != oldState {
		log.Printf("ChatId %d, state from %v to %v", f.ChatId, oldState, f.State)
	}
}
