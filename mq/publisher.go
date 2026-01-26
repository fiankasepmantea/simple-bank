package mq

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishUserCreated(
	ctx context.Context,
	ch *amqp.Channel,
	username string,
	userID int64,
) error {

	q, err := ch.QueueDeclare(
		"user.created",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	body, _ := json.Marshal(UserCreatedEvent{
		Username: username,
		UserID:   userID,
	})

	return ch.PublishWithContext(
		ctx,
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}
