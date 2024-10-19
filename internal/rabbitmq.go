package internal

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitClient struct {
	// The connection used by the client
	conn *amqp.Connection

	// Channel is used to process / Send messages
	ch *amqp.Channel
}

func ConnectRabbitMQ(username, password, host, vhost, caCert, clientCert, clientKey string) (*amqp.Connection, error) {
	ca, err := os.ReadFile(caCert)
	if err != nil {
		return nil, err
	}
	cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, err
	}

	rootCAs := x509.NewCertPool()

	rootCAs.AppendCertsFromPEM(ca)

	tlsCfg := &tls.Config{
		RootCAs:      rootCAs,
		Certificates: []tls.Certificate{cert},
	}

	// return amqp.Dial(fmt.Sprintf("amqps://%s:%s@%s/%s", username, password, host, vhost), tlsCfg)
	return amqp.DialTLS(fmt.Sprintf("amqps://%s:%s@%s/%s", username, password, host, vhost), tlsCfg)
}

func NewRabbitMQClient(conn *amqp.Connection) (RabbitClient, error) {
	ch, err := conn.Channel()
	if err != nil {
		return RabbitClient{}, err
	}

	if err := ch.Confirm(false); err != nil {
		return RabbitClient{}, err

	}

	return RabbitClient{
		conn: conn,
		ch:   ch,
	}, nil
}

func (rc RabbitClient) Close() error {
	return rc.ch.Close()
}

func (rc *RabbitClient) CreateQueue(queueName string, durable, autodelete bool) (amqp.Queue, error) {
	q, err := rc.ch.QueueDeclare(queueName, durable, autodelete, false, false, nil)
	if err != nil {
		return amqp.Queue{}, err
	}
	return q, err
}

// CreateBinding will bind the current channel to the given exchange using the routing key provided
func (rc RabbitClient) CreateBinding(name, binding, exchange string) error {
	// leaving no wait false, having nowait set to false will make the channel return an error if its false to bind
	return rc.ch.QueueBind(name, binding, exchange, false, nil)
}

// sends is used to pusblish payloads
func (rc RabbitClient) Send(ctx context.Context, exchange, routingKey string, options amqp.Publishing) error {
	confirmation, err := rc.ch.PublishWithDeferredConfirmWithContext(ctx,
		exchange,
		routingKey,
		// Mandatory is used to dertimine if an error should be returned upon failure
		true,
		false,
		options,
	)
	if err != nil {
		return err
	}
	log.Println(confirmation.Wait())
	return nil
}

// Consume is used to consume a queue
func (rc RabbitClient) Consume(queue, consumer string, autoAck bool) (<-chan amqp.Delivery, error) {
	return rc.ch.Consume(queue, consumer, autoAck, false, false, false, nil)
}

// Apply qos
// prefetch count - how many unackowleged messages the server can send
// prefetch size  - is int of how many bytes
// global 	      - Determines if the rule should be apllied globally or not

func (rc RabbitClient) ApplyQos(count, size int, global bool) error {
	return rc.ch.Qos(count, size, global)
}
