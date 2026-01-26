package mq

import (
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func NewRabbitMQ(url string) (*amqp.Connection, *amqp.Channel) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Println("⏳ waiting for rabbitmq...")
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("❌ cannot connect to rabbitmq: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("❌ cannot open channel: %v", err)
	}

	return conn, ch
}

