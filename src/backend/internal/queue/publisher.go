package queue

import (
	"context"
	"encoding/json"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"net"
	"sync"
	"time"
)

const (
	RegistrationQueue = "registration_queue"
	NotificationQueue = "notification_queue"
)

// Serialize connection replacement and confirms so callers cannot race with reconnect.
type Publisher struct {
	mu      sync.Mutex
	url     string
	conn    *amqp.Connection
	channel *amqp.Channel
	notif   chan *amqp.Error
	closed  bool
	done    chan struct{}
}

// IsConnected reports broker connectivity for readiness without exposing it.
func (p *Publisher) IsConnected() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return !p.closed && p.conn != nil && !p.conn.IsClosed() && p.channel != nil && !p.channel.IsClosed()
}

func NewPublisher(url string) (*Publisher, error) {
	p := &Publisher{url: url, done: make(chan struct{})}
	if err := p.connect(); err != nil {
		return nil, err
	}
	go p.handleReconnect()
	return p, nil
}
func (p *Publisher) connect() error {
	conn, err := amqp.DialConfig(p.url, amqp.Config{Dial: func(network, addr string) (net.Conn, error) { return net.DialTimeout(network, addr, 10*time.Second) }})
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}
	fail := func(err error) error { ch.Close(); conn.Close(); return err }
	for _, q := range []string{RegistrationQueue, NotificationQueue} {
		if _, err = ch.QueueDeclare(q, true, false, false, false, amqp.Table{"x-dead-letter-exchange": "", "x-dead-letter-routing-key": q + "_dlq"}); err != nil {
			return fail(err)
		}
		if _, err = ch.QueueDeclare(q+"_dlq", true, false, false, false, nil); err != nil {
			return fail(err)
		}
	}
	if err = ch.Confirm(false); err != nil {
		return fail(err)
	}
	notif := ch.NotifyClose(make(chan *amqp.Error, 1))
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return fail(fmt.Errorf("publisher closed"))
	}
	if p.conn != nil {
		p.conn.Close()
	}
	p.conn = conn
	p.channel = ch
	p.notif = notif
	return nil
}
func (p *Publisher) handleReconnect() {
	for {
		p.mu.Lock()
		notifications := p.notif
		closed := p.closed
		p.mu.Unlock()
		if closed {
			return
		}
		select {
		case <-p.done:
			return
		case _, ok := <-notifications:
			if !ok {
				select {
				case <-p.done:
					return
				default:
				}
			}
		}
		for {
			select {
			case <-p.done:
				return
			case <-time.After(2 * time.Second):
			}
			if err := p.connect(); err == nil {
				break
			} else {
				log.Printf("[RABBITMQ] reconnect: %v", err)
			}
		}
	}
}
func (p *Publisher) Publish(ctx context.Context, q string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.channel == nil || p.channel.IsClosed() {
		return fmt.Errorf("rabbitmq channel unavailable")
	}
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	confirm, err := p.channel.PublishWithDeferredConfirmWithContext(publishCtx, "", q, false, false, amqp.Publishing{ContentType: "application/json", Body: data, DeliveryMode: amqp.Persistent, Timestamp: time.Now()})
	if err != nil {
		return err
	}
	if confirm == nil {
		return fmt.Errorf("missing publisher confirmation")
	}
	acked, err := confirm.WaitContext(publishCtx)
	if err != nil {
		return err
	}
	if !acked {
		return fmt.Errorf("broker rejected message")
	}
	return nil
}
func (p *Publisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	close(p.done)
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
