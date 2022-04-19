package goals

import (
	"fmt"

	"github.com/dmage/gypd/api"
	"github.com/dmage/gypd/config"
	"github.com/dmage/gypd/statemanager"
)

type TaskSource struct {
	stateManager *statemanager.StateManager
}

func NewTaskSource(stateManager *statemanager.StateManager) *TaskSource {
	return &TaskSource{stateManager: stateManager}
}

func (ts *TaskSource) LoadTasks(cfg *config.Config) ([]*api.Task, error) {
	var tasks []*api.Task
	for _, goal := range ts.stateManager.GetGoals() {
		tasks = append(tasks, &api.Task{
			ID:      fmt.Sprintf("goal:%s", goal.ID),
			Summary: goal.ID,
			Labels: api.Labels{
				{Key: "_source", Value: "goal"},
			},
		})
	}
	return tasks, nil
}
