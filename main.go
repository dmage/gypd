package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dmage/gypd/api"
	"github.com/dmage/gypd/config"
	"github.com/dmage/gypd/goals"
	"github.com/dmage/gypd/mathgraph"
	"github.com/dmage/gypd/rh"
	"github.com/dmage/gypd/rhbz"
	"github.com/dmage/gypd/statemanager"
	"github.com/dmage/gypd/tasksource"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

//go:embed gypd-frontend/build
var frontend embed.FS

var (
	addr = flag.String("addr", ":8080", "http service address")
)

func updateMarkers(tasks []*api.Task, sm *statemanager.StateManager) {
	now := time.Now()
	for _, task := range tasks {
		taskState, ok := sm.GetTaskState(task.ID)
		if !ok {
			continue
		}

		if len(task.Labels.Get("parent")) == 0 && taskState.ParentID != "" {
			task.Labels.Add("parent", taskState.ParentID)
		}

		for _, marker := range taskState.Markers {
			if marker.Until == nil || marker.Until.After(now) {
				task.Labels.Add("marker", marker.Name)
			}
		}
	}
}

func reconsileTask(task *api.Task, team []config.TeamMember, scoreRules []config.ScoreRule) {
	assignee := task.Labels.Get("assignee")
	if len(assignee) == 1 && assignee[0] != api.AssigneeNone && assignee[0] != team[0].ID {
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
	task.Score = score

	task.Labels.Sort()
}

func updateTasks(tasks []*api.Task) []*api.Task {
	score := map[string]*mathgraph.Sum{}
	children := map[string]*mathgraph.Max{}
	for _, task := range tasks {
		score[task.ID] = mathgraph.NewSum()
		children[task.ID] = mathgraph.NewMax()
		score[task.ID].Add(mathgraph.NewConst(task.Score))
		score[task.ID].Add(children[task.ID])
	}
	for _, task := range tasks {
		for _, parent := range task.Labels.Get("parent") {
			if children[parent] != nil {
				children[parent].Add(score[task.ID])
			}
		}
	}

	for _, task := range tasks {
		task.Score = score[task.ID].Value()
		task.Labels.Add("score", strconv.Itoa(task.Score))
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Score > tasks[j].Score
	})

	return tasks
}

type Server struct {
	taskSource   tasksource.TaskSource
	stateManager *statemanager.StateManager
}

func (s *Server) urlParam(r *http.Request, name string) string {
	value := chi.URLParam(r, name)
	if r.URL.RawPath != "" {
		unescaped, err := url.PathUnescape(value)
		if err == nil {
			return unescaped
		}
	}
	return value
}

func (s *Server) getTasks() ([]*api.Task, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	tasks, err := s.taskSource.LoadTasks(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	updateMarkers(tasks, s.stateManager)
	for i := range tasks {
		reconsileTask(tasks[i], cfg.Team, cfg.ScoreRules)
	}
	tasks = updateTasks(tasks)

	return tasks, nil
}

func (s *Server) GetTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.getTasks()
	if err != nil {
		logrus.Errorf("Failed to get tasks: %v", err)
		http.Error(w, "Failed to get tasks", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (s *Server) PostTaskMarker(w http.ResponseWriter, r *http.Request) {
	id := s.urlParam(r, "id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	var params struct {
		Marker string `json:"marker"`
		Until  string `json:"until"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		logrus.Errorf("Failed to decode marker: %v", err)
		http.Error(w, "Failed to decode marker", http.StatusBadRequest)
		return
	}
	markerName := params.Marker
	if markerName == "" {
		http.Error(w, "missing marker", http.StatusBadRequest)
		return
	}
	var until *time.Time
	untilStr := params.Until
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

	err := s.stateManager.AddTaskMarker(id, config.Marker{
		Name:  markerName,
		Until: until,
	})
	if err != nil {
		logrus.Errorf("Failed to save marker: %v", err)
		http.Error(w, "Failed to save marker", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) PostTaskParent(w http.ResponseWriter, r *http.Request) {
	id := s.urlParam(r, "id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	var params struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		logrus.Errorf("Failed to decode request body: %v", err)
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}
	if params.ID == "" {
		http.Error(w, "missing parent id", http.StatusBadRequest)
		return
	}

	err := s.stateManager.SetTaskParent(id, params.ID)
	if err != nil {
		logrus.Errorf("Failed to save parent: %v", err)
		http.Error(w, "Failed to save parent", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) PostGoal(w http.ResponseWriter, r *http.Request) {
	var goal config.Goal
	if err := json.NewDecoder(r.Body).Decode(&goal); err != nil {
		logrus.Errorf("Failed to decode goal: %v", err)
		http.Error(w, "Failed to decode goal", http.StatusBadRequest)
		return
	}

	ok, err := s.stateManager.AddGoal(goal)
	if err != nil {
		logrus.Errorf("Failed to save goal: %v", err)
		http.Error(w, "Failed to save goal", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Goal already exists", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func main() {
	flag.Parse()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debug("Starting.")

	stateManager, err := statemanager.NewStateManager()
	if err != nil {
		logrus.Fatalf("Failed to initialize state: %v", err)
	}

	taskSource := tasksource.NewAggregated(
		tasksource.NewCached(rhbz.NewTaskSource(), 2*time.Minute),
		tasksource.NewCached(rh.NewTaskSource(), 5*time.Minute),
		goals.NewTaskSource(stateManager),
	)

	s := &Server{
		taskSource:   taskSource,
		stateManager: stateManager,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/api/tasks", s.GetTasks)
	r.Post("/api/tasks/{id}/markers", s.PostTaskMarker)
	r.Post("/api/tasks/{id}/parent", s.PostTaskParent)
	r.Post("/api/goals", s.PostGoal)

	staticFS, err := fs.Sub(frontend, "gypd-frontend/build")
	if err != nil {
		logrus.Fatalf("Failed to initialize static file system: %v", err)
	} else {
		r.Mount("/", http.FileServer(http.FS(staticFS)))
	}

	err = http.ListenAndServe(*addr, r)
	if err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}
