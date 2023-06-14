// A homegrown functional programming library, heavily using go generics.
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

func Any[T any](a []T, f func(T) bool) bool {
	for _, e := range a {
		if f(e) {
			return true
		}
	}
	return false
}

func All[T any](a []T, f func(T) bool) bool {
	for _, e := range a {
		if !f(e) {
			return false
		}
	}
	return true
}
