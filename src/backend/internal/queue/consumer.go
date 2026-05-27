package queue

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer manages RabbitMQ consuming
type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewConsumer(url string) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS (prefetch count = 1 for fair dispatch)
	if err := ch.Qos(1, 0, false); err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	log.Println("[RABBITMQ] Consumer connected")
	return &Consumer{conn: conn, channel: ch}, nil
}

// Consume starts consuming messages from the specified queue
func (c *Consumer) Consume(queueName string) (<-chan amqp.Delivery, error) {
	msgs, err := c.channel.Consume(
		queueName,
		"",    // consumer tag (auto-generated)
		false, // auto-ack = false (manual acknowledgment)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to consume from %s: %w", queueName, err)
	}

	return msgs, nil
}

func (c *Consumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
