package mathgraph

type Value interface {
	Value(visited ...Value) int
}

type Const struct {
	value int
}

func NewConst(c int) Const {
	return Const{
		value: c,
	}
}

func (c Const) Value(visited ...Value) int {
	return c.value
}

type Max struct {
	values []Value
}

func NewMax(values ...Value) *Max {
	return &Max{
		values: values,
	}
}

func (m *Max) Add(other Value) {
	m.values = append(m.values, other)
}

func (m *Max) Value(visited ...Value) int {
	if m == nil || len(m.values) == 0 {
		return 0
	}
	for _, v := range visited {
		if v == m {
			return 0
		}
	}
	visited = append(visited, m)
	max := m.values[0].Value(visited...)
	for _, val := range m.values[1:] {
		v := val.Value(visited...)
		if v > max {
			max = v
		}
	}
	return max
}

type Sum struct {
	values []Value
}

func NewSum(values ...Value) *Sum {
	return &Sum{
		values: values,
	}
}

func (s *Sum) Add(other Value) {
	s.values = append(s.values, other)
}

func (s *Sum) Value(visited ...Value) int {
	if s == nil || len(s.values) == 0 {
		return 0
	}
	for _, v := range visited {
		if v == s {
			return 0
		}
	}
	visited = append(visited, s)
	sum := 0
	for _, val := range s.values {
		sum += val.Value(visited...)
	}
	return sum
}
