package main

import (
	"benjaminarmijo3/eventdrivenrabbit/internal"
	"context"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"golang.org/x/sync/errgroup"
)

func main() {

	conn, err := internal.ConnectRabbitMQ("benja", "secret", "localhost:5671", "customers",
		"/Users/benja/clases/go/rabbit/tls-gen/basic/result/ca_certificate.pem",
		"/Users/benja/clases/go/rabbit/tls-gen/basic/result/client_benja_certificate.pem",
		"/Users/benja/clases/go/rabbit/tls-gen/basic/result/client_benja_key.pem",
	)
	if err != nil {
		panic(err)
	}

	publishConn, err := internal.ConnectRabbitMQ("benja", "secret", "localhost:5671", "customers",
		"/Users/benja/clases/go/rabbit/tls-gen/basic/result/ca_certificate.pem",
		"/Users/benja/clases/go/rabbit/tls-gen/basic/result/client_benja_certificate.pem",
		"/Users/benja/clases/go/rabbit/tls-gen/basic/result/client_benja_key.pem",
	)
	if err != nil {
		panic(err)
	}

	defer publishConn.Close()

	client, err := internal.NewRabbitMQClient(conn)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	publishClient, err := internal.NewRabbitMQClient(publishConn)
	if err != nil {
		panic(err)
	}
	defer publishClient.Close()

	queue, err := client.CreateQueue("", true, true)
	if err != nil {
		panic(err)
	}

	if err := client.CreateBinding(queue.Name, "", "customer_events"); err != nil {
		panic(err)
	}

	messageBus, err := client.Consume(queue.Name, "email-service", false)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	//err group alow us concurrent tasks

	if err := client.ApplyQos(10, 0, true); err != nil {
		panic(err)
	}

	var blocking chan struct{}

	// go func() {
	g.SetLimit(10)

	go func() {
		for message := range messageBus {
			msg := message
			g.Go(func() error {
				log.Printf("New message: %v", msg)
				time.Sleep(10 * time.Second)
				if err := msg.Ack(false); err != nil {
					log.Println("Ack message failed")
					return err
				}

				if err := publishClient.Send(ctx, "customer_callbacks", msg.ReplyTo, amqp091.Publishing{
					ContentType:   "text/plain",
					DeliveryMode:  amqp091.Persistent,
					Body:          []byte(`RPC COMPLETE`),
					CorrelationId: msg.CorrelationId,
				}); err != nil {
					panic(err)
				}
				log.Printf("Ackownledge message %s\n", message.MessageId)
				return nil
			})
		}
	}()

	// var blocking chan struct{}

	// go func() {
	// 	for message := range messageBus {
	// 		log.Printf("New Message: %v", message)
	// 		if !message.Redelivered {
	// 			message.Nack(false, true)
	// 			continue
	// 		}
	// 		if err := message.Ack(false); err != nil {
	// 			log.Println("Acknowledge meessage failed")
	// 			continue
	// 		}
	// 		// if err := message.
	// 		log.Printf("Aknowledge message %s\n", message.MessageId)
	// 	}
	// }()

	log.Println("Consuming, to close the program press CTRL+C")
	<-blocking
}
