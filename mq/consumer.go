package mq

import (
	"context"
	"encoding/json"
	"log"
	db "simple-bank/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	amqp "github.com/rabbitmq/amqp091-go"
)

func StartUserCreatedConsumer(
	ch *amqp.Channel,
	store db.Store,
) {

	q, err := ch.QueueDeclare(
		"user.created",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("🐇 RabbitMQ consumer started: user.created")

	go func() {
		for msg := range msgs {
			var event UserCreatedEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				msg.Nack(false, false)
				continue
			}

			_, err := store.UpdateUserID(
				context.Background(),
				db.UpdateUserIDParams{
					Username: event.Username,
					ID: pgtype.Int8{
						Int64: event.UserID,
						Valid: true,
					},
				},
			)

			if err != nil {
				log.Println("failed update user id:", err)
				msg.Nack(false, true)
				continue
			}

			msg.Ack(false)
		}

	}()
}
