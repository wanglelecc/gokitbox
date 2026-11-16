package consumer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type amqpSession struct {
	conn             *amqp.Connection
	channel          *amqp.Channel
	wire             net.Conn
	messages         <-chan amqp.Delivery
	returned         <-chan amqp.Return
	closed           <-chan *amqp.Error
	stopCancellation func() bool
	once             sync.Once
}

func openRabbitSession(parent context.Context, config *RabbitmqConfig, options RuntimeOptions) (session rabbitSession, err error) {
	setup, cancel := boundedContext(parent, options.OperationTimeout)
	defer cancel()
	var wire net.Conn
	var stopSetup func() bool
	var setupFinished chan struct{}
	// The SDK's PublishWithContext only checks before sending. We own the socket
	// so setup and later operation deadlines can actually interrupt blocked I/O.
	conn, err := amqp.DialConfig(config.URL, amqp.Config{Heartbeat: 10 * time.Second, Locale: "en_US", Dial: func(network, address string) (net.Conn, error) {
		var err error
		wire, err = (&net.Dialer{Timeout: options.OperationTimeout}).DialContext(setup, network, address)
		if err != nil {
			return nil, err
		}
		deadline, _ := setup.Deadline()
		if err = wire.SetDeadline(deadline); err != nil {
			wire.Close()
			return nil, err
		}
		setupFinished = make(chan struct{})
		stopSetup = context.AfterFunc(setup, func() { wire.Close(); close(setupFinished) })
		return wire, nil
	}})
	defer func() {
		if stopSetup != nil && !stopSetup() {
			<-setupFinished
		}
		if err != nil && wire != nil {
			wire.Close()
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("rabbitmq connect: %w", err)
	}
	s := &amqpSession{conn: conn, wire: wire, closed: conn.NotifyClose(make(chan *amqp.Error, 1))}
	defer func() {
		if err != nil {
			s.Close()
		}
	}()
	s.channel, err = conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}
	if err = s.channel.Qos(config.PrefetchCount, 0, false); err != nil {
		return nil, fmt.Errorf("rabbitmq qos: %w", err)
	}
	if config.FailExchange != "" {
		if err = s.channel.Confirm(false); err != nil {
			return nil, fmt.Errorf("rabbitmq confirm mode: %w", err)
		}
		s.returned = s.channel.NotifyReturn(make(chan amqp.Return, 1))
	}
	s.messages, err = s.channel.Consume(config.ConsumerQueue, "", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq consume: %w", err)
	}
	if err = setup.Err(); err != nil {
		return nil, err
	}
	if stopSetup != nil {
		if !stopSetup() {
			<-setupFinished
			stopSetup = nil
			return nil, setup.Err()
		}
		stopSetup = nil
	}
	if err = wire.SetDeadline(time.Time{}); err != nil {
		return nil, err
	}
	s.stopCancellation = context.AfterFunc(parent, func() { wire.Close() })
	return s, nil
}
func (s *amqpSession) deliveries() <-chan amqp.Delivery { return s.messages }
func (s *amqpSession) operation(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	deadline, _ := ctx.Deadline()
	if err := s.wire.SetWriteDeadline(deadline); err != nil {
		return err
	}
	finished := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { s.wire.Close(); close(finished) })
	err := fn()
	if !stop() {
		<-finished
	}
	if contextErr := ctx.Err(); contextErr != nil {
		return contextErr
	}
	if err != nil {
		return err
	}
	return s.wire.SetWriteDeadline(time.Time{})
}
func (s *amqpSession) transferFailure(ctx context.Context, exchange string, event amqp.Delivery) error {
	return s.operation(ctx, func() error {
		// Preserve identity and delivery properties. Durability still requires an
		// appropriately configured target queue and persistent source messages.
		message := amqp.Publishing{Headers: event.Headers, ContentType: event.ContentType, ContentEncoding: event.ContentEncoding, DeliveryMode: event.DeliveryMode, Priority: event.Priority, CorrelationId: event.CorrelationId, ReplyTo: event.ReplyTo, Expiration: event.Expiration, MessageId: event.MessageId, Timestamp: event.Timestamp, Type: event.Type, AppId: event.AppId, Body: event.Body}
		confirm, err := s.channel.PublishWithDeferredConfirmWithContext(ctx, exchange, event.RoutingKey, true, false, message)
		if err != nil {
			return err
		}
		if confirm == nil {
			return errors.New("rabbitmq: publisher confirm not enabled")
		}
		acknowledged, err := confirm.WaitContext(ctx)
		if err != nil {
			return err
		}
		if !acknowledged {
			return errors.New("rabbitmq: failure publish nacked")
		}
		// AMQP orders basic.return before confirm. The SDK dispatches them in wire
		// order; with one outstanding publish a buffered return is visible here.
		select {
		case returned, ok := <-s.returned:
			if !ok {
				return errors.New("rabbitmq: return channel closed")
			}
			return fmt.Errorf("rabbitmq: failure publish unroutable (%d)", returned.ReplyCode)
		default:
			return nil
		}
	})
}
func (s *amqpSession) settle(ctx context.Context, event amqp.Delivery, reject, requeue bool) error {
	return s.operation(ctx, func() error {
		if reject {
			return event.Reject(requeue)
		}
		return event.Ack(false)
	})
}
func (s *amqpSession) Close() error {
	var result error
	s.once.Do(func() {
		if s.stopCancellation != nil {
			s.stopCancellation()
		}
		if s.wire != nil {
			result = s.wire.Close()
		} else if s.conn != nil {
			result = s.conn.Close()
		}
		if s.closed != nil {
			for range s.closed {
			}
		}
	})
	if errors.Is(result, net.ErrClosed) {
		return nil
	}
	return result
}
