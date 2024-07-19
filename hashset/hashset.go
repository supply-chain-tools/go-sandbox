package hashset

type Set[T comparable] interface {
	Add(element ...T)
	Contains(element T) bool
	Values() []T
	Size() int
	Difference(other Set[T]) Set[T]
}

type hashSet[T comparable] struct {
	elements map[T]struct{}
}

func New[T comparable](elements ...T) Set[T] {
	result := &hashSet[T]{
		elements: make(map[T]struct{}),
	}

	for _, element := range elements {
		result.Add(element)
	}

	return result
}

func (hs *hashSet[T]) Add(element ...T) {
	for _, element := range element {
		(*hs).elements[element] = struct{}{}
	}
}

func (hs *hashSet[T]) Contains(element T) bool {
	_, found := (*hs).elements[element]
	return found
}

func (hs *hashSet[T]) Values() []T {
	result := make([]T, 0, len((*hs).elements))
	for k := range (*hs).elements {
		result = append(result, k)
	}

	return result
}

func (hs *hashSet[T]) Size() int {
	return len((*hs).elements)
}

func (hs *hashSet[T]) Difference(other Set[T]) Set[T] {
	result := New[T]()
	for element := range (*hs).elements {
		if !other.Contains(element) {
			result.Add(element)
		}
	}

	return result
}
