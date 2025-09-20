package messaging

type MessageBrokerInterface interface {
	Publish(subject, msg string) error
}
