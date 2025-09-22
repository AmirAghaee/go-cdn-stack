package messaging

type MessageBrokerInterface interface {
	Publish(subject, msg string) error
	Subscribe(subject string, handler func(msg string)) error
}
