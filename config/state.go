package config

import (
	"io/ioutil"
	"os"
	"time"

	"sigs.k8s.io/yaml"
)

type Goal struct {
	ID    string `json:"id"`
	Score int    `json:"score"`
}

type Marker struct {
	Name  string     `json:"name"`
	Until *time.Time `json:"until,omitempty"`
}

type TaskState struct {
	ID       string   `json:"id"`
	ParentID string   `json:"parent_id,omitempty"`
	Markers  []Marker `json:"markers,omitempty"`
}

type State struct {
	Goals []Goal      `json:"goals,omitempty"`
	Tasks []TaskState `json:"tasks,omitempty"`
}

func LoadState() (*State, error) {
	var state State
	buf, err := ioutil.ReadFile("./state.yaml")
	if os.IsNotExist(err) {
		return &state, nil
	} else if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(buf, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func SaveState(state *State) error {
	buf, err := yaml.Marshal(state)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("./state.yaml", buf, 0644)
}
