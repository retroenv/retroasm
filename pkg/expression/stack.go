package expression

type stack[T any] struct {
	data []T
}

func (st *stack[T]) len() int {
	return len(st.data)
}

func (st *stack[T]) push(item T) {
	st.data = append(st.data, item)
}

func (st *stack[T]) pop() T {
	lastIdx := len(st.data) - 1
	item := st.data[lastIdx]
	st.data = st.data[:lastIdx]
	return item
}

func (st *stack[T]) last() T {
	lastIdx := len(st.data) - 1
	return st.data[lastIdx]
}
