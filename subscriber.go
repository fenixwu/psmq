package psmq

import (
	"log"

	"github.com/streadway/amqp"
)

// Handler 接到訊息之後的動作
type Handler func(data []byte) error

// Subscriber 接收端
type Subscriber struct {
	psmq    *psmq
	msgs    <-chan amqp.Delivery
	handler Handler
}

// NewSubscriber new a subscriber
func NewSubscriber(pb *psmq, exchange string, queueTTLSec int32, h Handler) (*Subscriber, error) {
	failedPrefix := "New subscriber failed"
	err := pb.declareExchange(exchange)
	if err != nil {
		return nil, failedError(failedPrefix, err)
	}

	queue, err := pb.declareQueue(queueTTLSec)
	if err != nil {
		return nil, failedError(failedPrefix, err)
	}

	// ----- Binding Queue -----
	err = pb.bindQueue(queue, exchange)
	if err != nil {
		return nil, failedError(failedPrefix, err)
	}

	// ----- Consumer -----
	msgs, err := pb.channel.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return nil, failedError(failedPrefix, err)
	}
	return &Subscriber{pb, msgs, h}, nil
}

// Run a subscriber
func (s *Subscriber) Run() {
	forever := make(chan bool)

	go func() {
		for d := range s.msgs {
			err := s.handler(d.Body)
			if err != nil {
				s.psmq.channel.Ack(d.DeliveryTag, false)
				continue
			}
			s.psmq.channel.Nack(d.DeliveryTag, false, false)
		}
	}()
	log.Printf("[psmq] Waiting for message. Press \"CTRL+C\" to exit.")
	<-forever
}
