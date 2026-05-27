package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	RegistrationQueue = "registration_queue"
	NotificationQueue = "notification_queue"
)

// Publisher manages RabbitMQ publishing with auto-reconnect
type Publisher struct {
	url      string
	conn     *amqp.Connection
	channel  *amqp.Channel
	notif    chan *amqp.Error
	isClosed bool
}

func NewPublisher(url string) (*Publisher, error) {
	p := &Publisher{url: url}
	if err := p.connect(); err != nil {
		return nil, err
	}
	go p.handleReconnect()
	return p, nil
}

func (p *Publisher) connect() error {
	conn, err := amqp.Dial(p.url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	// Declare queues
	for _, q := range []string{RegistrationQueue, NotificationQueue} {
		_, err := ch.QueueDeclare(q, true, false, false, false, amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": q + "_dlq",
		})
		if err != nil {
			return err
		}
		ch.QueueDeclare(q+"_dlq", true, false, false, false, nil)
	}

	p.conn = conn
	p.channel = ch
	p.notif = make(chan *amqp.Error)
	p.channel.NotifyClose(p.notif)

	log.Println("[RABBITMQ] Publisher connected")
	return nil
}

func (p *Publisher) handleReconnect() {
	for {
		if p.isClosed {
			return
		}

		err := <-p.notif
		if err != nil {
			log.Printf("[RABBITMQ] Connection lost, reconnecting... (%v)", err)
			for {
				time.Sleep(2 * time.Second)
				if err := p.connect(); err == nil {
					log.Println("[RABBITMQ] Reconnected successfully")
					break
				}
				log.Println("[RABBITMQ] Reconnect failed, retrying...")
			}
		}
	}
}

func (p *Publisher) Publish(ctx context.Context, queueName string, message interface{}) error {
	if p.channel == nil || p.channel.IsClosed() {
		return fmt.Errorf("rabbitmq channel is closed")
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.channel.PublishWithContext(publishCtx,
		"", queueName, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
}

func (p *Publisher) Close() {
	p.isClosed = true
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
