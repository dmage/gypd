package api

import "sort"

type Status string

const (
	StatusNew      Status = "NEW"
	StatusAssigned Status = "ASSIGNED"
	StatusOnDev    Status = "ON_DEV"
	StatusPost     Status = "POST"
	StatusModified Status = "MODIFIED"
	StatusOnQA     Status = "ON_QA"
	StatusVerified Status = "VERIFIED"
	StatusClosed   Status = "CLOSED"
)

func (s Status) String() string {
	return string(s)
}

type Priority string

const (
	PriorityP1 Priority = "P1"
	PriorityP2 Priority = "P2"
	PriorityP3 Priority = "P3"
	PriorityP4 Priority = "P4"
	PriorityP5 Priority = "P5"
)

func (p Priority) String() string {
	return string(p)
}

const (
	AssigneeNone string = "NONE"
)

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Labels []KeyValue

func (l Labels) Has(key, value string) bool {
	for _, label := range l {
		if label.Key == key && label.Value == value {
			return true
		}
	}
	return false
}

func (l *Labels) Add(key, value string) {
	if l.Has(key, value) {
		return
	}
	*l = append(*l, KeyValue{Key: key, Value: value})
}

func (l *Labels) Get(key string) []string {
	var values []string
	for _, label := range *l {
		if label.Key == key {
			values = append(values, label.Value)
		}
	}
	return values
}

func (l *Labels) Sort() {
	if l == nil {
		return
	}
	sort.Slice(*l, func(i, j int) bool {
		if (*l)[i].Key != (*l)[j].Key {
			return (*l)[i].Key < (*l)[j].Key
		}
		return (*l)[i].Value < (*l)[j].Value
	})
}

func (l Labels) DeepCopy() Labels {
	clone := make(Labels, len(l))
	copy(clone, l)
	return clone
}

type Task struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Summary string `json:"summary"`
	Labels  Labels `json:"labels"`
	Score   int    `json:"score"`
}

func (t *Task) DeepCopy() *Task {
	return &Task{
		ID:      t.ID,
		URL:     t.URL,
		Summary: t.Summary,
		Labels:  t.Labels.DeepCopy(),
	}
}
