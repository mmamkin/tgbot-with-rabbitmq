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
	amqpDSN := os.Getenv("AMQP_DSN")

	cfg := WorkerCfg{
		AmqpDSN:     amqpDSN,
		SendQ:       "queue2",
		RecvQ:       "queue1",
		ConsumerTag: "worker",
	}
	worker := NewWorker(cfg)
	err := worker.Start()
	if err != nil {
		log.Panicf("Error starting worker: %s", err)
	}
	<-exit
	fmt.Println("Stopping...")
	worker.Stop()
}
