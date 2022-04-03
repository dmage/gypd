package api

import (
	"encoding/json"
	"sort"
)

type StringSet map[string]struct{}

func (s StringSet) Has(key string) bool {
	if s == nil {
		return false
	}
	_, ok := s[key]
	return ok
}

func (s *StringSet) Add(key string) {
	if *s == nil {
		*s = make(map[string]struct{})
	}
	(*s)[key] = struct{}{}
}

func (s StringSet) Remove(key string) {
	delete(s, key)
}

func (s StringSet) Sorted() []string {
	keys := []string{}
	for key := range s {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *StringSet) Reset() {
	*s = nil
}

func (s StringSet) Empty() bool {
	return len(s) == 0
}

func (s StringSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Sorted())
}

func (s *StringSet) UnmarshalJSON(data []byte) error {
	var keys []string
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}
	if len(keys) == 0 {
		*s = nil
		return nil
	}
	*s = make(map[string]struct{}, len(keys))
	for _, key := range keys {
		(*s)[key] = struct{}{}
	}
	return nil
}
