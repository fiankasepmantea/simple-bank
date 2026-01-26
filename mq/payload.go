package mq

type UserCreatedEvent struct {
	Username string `json:"username"`
	UserID   int64  `json:"user_id"`
}
