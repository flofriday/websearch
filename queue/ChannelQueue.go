package queue

import "errors"

type ChannelQueue[T any] struct {
	channel chan T
}

func NewChannelQueue[T any](channel chan T) *ChannelQueue[T] {
	return &ChannelQueue[T]{
		channel: channel,
	}
}

func (q *ChannelQueue[T]) Put(item T) error {
	q.channel <- item
	return nil
}

func (q *ChannelQueue[T]) Get() (T, error) {

	item, ok := <-q.channel
	if !ok {
		return item, errors.New("Channel closed")
	}
	return item, nil
}

func (q *ChannelQueue[T]) Size() (int64, error) {
	return int64(len(q.channel)), nil
}

func (q *ChannelQueue[T]) Close() {
	close(q.channel)
}
