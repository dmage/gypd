package statemanager

import (
	"github.com/dmage/gypd/config"
)

type StateManager struct {
	state *config.State
}

func NewStateManager() (*StateManager, error) {
	state, err := config.LoadState()
	if err != nil {
		return nil, err
	}
	return &StateManager{
		state: state,
	}, nil
}

func (sm *StateManager) flush() error {
	return config.SaveState(sm.state)
}

func (sm *StateManager) GetGoals() []config.Goal {
	return sm.state.Goals
}

func (sm *StateManager) GetTaskState(id string) (config.TaskState, bool) {
	for _, taskState := range sm.state.Tasks {
		if taskState.ID == id {
			return taskState, true
		}
	}
	return config.TaskState{}, false
}

func (sm *StateManager) getOrCreateTaskState(id string) *config.TaskState {
	var taskState *config.TaskState
	for i, ts := range sm.state.Tasks {
		if ts.ID == id {
			taskState = &sm.state.Tasks[i]
			break
		}
	}
	if taskState == nil {
		sm.state.Tasks = append(sm.state.Tasks, config.TaskState{ID: id})
		taskState = &sm.state.Tasks[len(sm.state.Tasks)-1]
	}
	return taskState
}

func (sm *StateManager) AddTaskMarker(taskID string, marker config.Marker) error {
	taskState := sm.getOrCreateTaskState(taskID)

	var taskMarker *config.Marker
	for i, m := range taskState.Markers {
		if m.Name == marker.Name {
			taskMarker = &taskState.Markers[i]
		}
	}
	if taskMarker == nil {
		taskState.Markers = append(taskState.Markers, config.Marker{Name: marker.Name})
		taskMarker = &taskState.Markers[len(taskState.Markers)-1]
	}

	taskMarker.Until = marker.Until

	return sm.flush()
}

func (sm *StateManager) SetTaskParent(taskID string, parentID string) error {
	taskState := sm.getOrCreateTaskState(taskID)

	taskState.ParentID = parentID

	return sm.flush()
}

func (sm *StateManager) AddGoal(goal config.Goal) (bool, error) {
	for _, g := range sm.state.Goals {
		if g.ID == goal.ID {
			return false, nil
		}
	}
	sm.state.Goals = append(sm.state.Goals, goal)
	return true, sm.flush()
}
