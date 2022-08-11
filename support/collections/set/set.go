package set

type Set[T comparable] map[T]struct{}

func NewSet[T comparable](capacity int) Set[T] {
	return make(map[T]struct{}, capacity)
}

func (set Set[T]) Add(item T) {
	set[item] = struct{}{}
}

func (set Set[T]) AddSlice(items []T) {
	for _, item := range items {
		set[item] = struct{}{}
	}
}

func (set Set[T]) Remove(item T) {
	delete(set, item)
}

func (set Set[T]) Contains(item T) bool {
	_, ok := set[item]
	return ok
}
