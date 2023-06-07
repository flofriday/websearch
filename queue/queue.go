package queue

type Queue[T any] interface {
	Put(item T) error
	Get() (T, error)
	Size() (int64, error)
	Close()
}
