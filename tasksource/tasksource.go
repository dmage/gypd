package tasksource

import (
	"time"

	"github.com/dmage/gypd/api"
	"github.com/dmage/gypd/config"
)

type TaskSource interface {
	LoadTasks(config *config.Config) ([]*api.Task, error)
}

type Aggregated struct {
	sources []TaskSource
}

func NewAggregated(sources ...TaskSource) *Aggregated {
	return &Aggregated{sources: sources}
}

func (a *Aggregated) LoadTasks(cfg *config.Config) ([]*api.Task, error) {
	var tasks []*api.Task
	for _, s := range a.sources {
		ts, err := s.LoadTasks(cfg)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, ts...)
	}
	return tasks, nil
}

type Cached struct {
	source TaskSource
	ttl    time.Duration

	tasks      []*api.Task
	validUntil time.Time
}

func NewCached(source TaskSource, ttl time.Duration) *Cached {
	return &Cached{
		source: source,
		ttl:    ttl,
	}
}

func (c *Cached) tasksDeepCopy() []*api.Task {
	tasks := make([]*api.Task, len(c.tasks))
	for i, task := range c.tasks {
		tasks[i] = task.DeepCopy()
	}
	return tasks
}

func (c *Cached) LoadTasks(cfg *config.Config) ([]*api.Task, error) {
	if time.Now().Before(c.validUntil) {
		return c.tasksDeepCopy(), nil
	}
	tasks, err := c.source.LoadTasks(cfg)
	if err != nil {
		return tasks, err
	}
	c.tasks = tasks
	c.validUntil = time.Now().Add(c.ttl)
	return c.tasksDeepCopy(), nil
}
