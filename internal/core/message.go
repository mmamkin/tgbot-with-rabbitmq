package core

type Message struct {
	BotType string
	ChatId  string
	Text    string
	Command string
}

type Config struct {
	SendQ       string
	RecvQ       string
	AmqpDSN     string
	ConsumerTag string
}
