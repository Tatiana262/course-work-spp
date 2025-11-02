package rabbitmq_consumer

import "context"

type Consumer interface {
	StartConsuming(ctx context.Context) error
	Close() error
}