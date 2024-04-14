package delayqueue

type Option func(*delayQueue)

func WithConcurrent(concurrent int) Option {
	return func(queue *delayQueue) {
		queue.Concurrent = concurrent
	}
}

func WithSerializer(serializer Serializer) Option {
	return func(queue *delayQueue) {
		queue.Serializer = serializer
		setDefaultSerializer(serializer)
	}
}
