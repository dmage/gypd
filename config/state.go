package config

import (
	"io/ioutil"
	"os"
	"time"

	"sigs.k8s.io/yaml"
)

type Marker struct {
	Name  string     `json:"name"`
	Until *time.Time `json:"until,omitempty"`
}

type TaskState struct {
	ID      string   `json:"id"`
	Markers []Marker `json:"markers"`
}

type State struct {
	Tasks []TaskState `json:"tasks"`
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
