package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dmage/gypd/api"
	"github.com/dmage/gypd/config"
	"github.com/dmage/gypd/rh"
	"github.com/dmage/gypd/rhbz"
	"github.com/sirupsen/logrus"
)

type TaskSource interface {
	LoadTasks(config *config.Config) ([]*api.Task, error)
}

type cachedTaskSource struct {
	source TaskSource
	ttl    time.Duration

	tasks      []*api.Task
	validUntil time.Time
}

func newCachedTaskSource(source TaskSource, ttl time.Duration) TaskSource {
	return &cachedTaskSource{
		source: source,
		ttl:    ttl,
	}
}

func (c *cachedTaskSource) tasksDeepCopy() []*api.Task {
	tasks := make([]*api.Task, len(c.tasks))
	for i, task := range c.tasks {
		tasks[i] = task.DeepCopy()
	}
	return tasks
}

func (c *cachedTaskSource) LoadTasks(cfg *config.Config) ([]*api.Task, error) {
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

func findTaskState(state *config.State, id string) (config.TaskState, bool) {
	for _, taskState := range state.Tasks {
		if taskState.ID == id {
			return taskState, true
		}
	}
	return config.TaskState{}, false
}

func updateMarkers(tasks []*api.Task, state *config.State) error {
	now := time.Now()

	for _, task := range tasks {
		taskState, ok := findTaskState(state, task.ID)
		if !ok {
			continue
		}

		for _, marker := range taskState.Markers {
			if marker.Until == nil || marker.Until.After(now) {
				task.Labels.Add("marker", marker.Name)
			}
		}
	}
	return nil
}

func reconsileTask(task *api.Task, team []config.TeamMember, scoreRules []config.ScoreRule) {
	assignee := task.Labels.Get("assignee")
	if len(assignee) != 1 {
		task.Labels.Add("flag", "incorrect")
		task.Labels.Add("error", "bad assignee")
	} else if assignee[0] != api.AssigneeNone && assignee[0] != team[0].ID {
		task.Labels.Add("flag", "delegated")
	}

	if len(task.Labels.Get("blocked-by")) > 0 {
		task.Labels.Add("flag", "blocked")
	}

	if task.Labels.Has("marker", "blocked") {
		task.Labels.Add("flag", "blocked")
	}

	score := 0
	for _, rule := range scoreRules {
		if rule.Match(task.Labels) {
			score += rule.Score
		}
	}
	if score < 0 {
		score = 0
	}
	task.Score = score
	task.Labels.Add("score", strconv.Itoa(score))

	task.Labels.Sort()
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debug("Starting.")

	taskSources := []TaskSource{
		newCachedTaskSource(rhbz.NewTaskSource(), 2*time.Minute),
		newCachedTaskSource(rh.NewTaskSource(), 5*time.Minute),
	}

	http.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		cfg, err := config.LoadConfig()
		if err != nil {
			logrus.Errorf("Failed to load config: %v", err)
			http.Error(w, "Failed to load config", http.StatusInternalServerError)
			return
		}

		state, err := config.LoadState()
		if err != nil {
			logrus.Errorf("Failed to load state: %v", err)
			http.Error(w, "Failed to load state", http.StatusInternalServerError)
			return
		}

		var tasks []*api.Task
		for _, ts := range taskSources {
			t, err := ts.LoadTasks(cfg)
			if err != nil {
				logrus.Errorf("Failed to load tasks: %v", err)
				http.Error(w, "Failed to load tasks", http.StatusInternalServerError)
				return
			}
			tasks = append(tasks, t...)
		}

		if err := updateMarkers(tasks, state); err != nil {
			logrus.Errorf("Failed to update markers: %v", err)
			http.Error(w, "Failed to update markers", http.StatusInternalServerError)
			return
		}

		for i := range tasks {
			reconsileTask(tasks[i], cfg.Team, cfg.ScoreRules)
		}

		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].Score > tasks[j].Score
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	})
	http.HandleFunc("/api/add-marker", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		markerName := r.URL.Query().Get("marker")
		if markerName == "" {
			http.Error(w, "missing marker", http.StatusBadRequest)
			return
		}
		var until *time.Time
		untilStr := r.URL.Query().Get("until")
		if untilStr != "" {
			if strings.HasPrefix(untilStr, "+") && strings.HasSuffix(untilStr, "h") {
				d, err := strconv.Atoi(untilStr[1 : len(untilStr)-1])
				if err != nil {
					http.Error(w, "invalid until", http.StatusBadRequest)
					return
				}
				untilTime := time.Now().Add(time.Duration(d) * time.Hour)
				until = &untilTime
			} else {
				untilTime, err := time.Parse(time.RFC3339, untilStr)
				if err != nil {
					http.Error(w, "invalid until", http.StatusBadRequest)
					return
				}
				until = &untilTime
			}
		}
		state, err := config.LoadState()
		if err != nil {
			logrus.Errorf("Failed to load state: %s", err)
			http.Error(w, "failed to load state", http.StatusInternalServerError)
			return
		}
		var taskState *config.TaskState
		for i, ts := range state.Tasks {
			if ts.ID == id {
				taskState = &state.Tasks[i]
				break
			}
		}
		if taskState == nil {
			state.Tasks = append(state.Tasks, config.TaskState{ID: id})
			taskState = &state.Tasks[len(state.Tasks)-1]
		}
		var marker *config.Marker
		for i, m := range taskState.Markers {
			if m.Name == markerName {
				marker = &taskState.Markers[i]
			}
		}
		if marker == nil {
			taskState.Markers = append(taskState.Markers, config.Marker{Name: markerName})
			marker = &taskState.Markers[len(taskState.Markers)-1]
		}
		marker.Until = until
		if err := config.SaveState(state); err != nil {
			logrus.Errorf("Failed to save state: %s", err)
			http.Error(w, "failed to save state", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	http.ListenAndServe(":8080", nil)
}
