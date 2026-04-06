package pool

import rootpool "github.com/squaredbusinessman/go-musthave-metrics"

type Resetter = rootpool.Resetter

type Pool[T Resetter] = rootpool.Pool[T]

func New[T Resetter]() *Pool[T] {
	return rootpool.New[T]()
}
