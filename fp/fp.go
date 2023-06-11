package fp

func Map[T any, M any](a []T, f func(T) M) []M {
	l := make([]M, len(a))
	for i, e := range a {
		l[i] = f(e)
	}
	return l
}

func MapErr[T any, M any](a []T, f func(T) (M, error)) ([]M, error) {
	var err error
	l := make([]M, len(a))
	for i, e := range a {
		l[i], err = f(e)
		if err != nil {
			return nil, err
		}
	}
	return l, nil
}
