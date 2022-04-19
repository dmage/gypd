package rhbz

import (
	"fmt"
	"strings"

	"github.com/dmage/gypd/api"
	"github.com/dmage/gypd/config"
	"github.com/eparis/bugzilla"
	"github.com/sirupsen/logrus"
)

var (
	bugzillaEndpoint = "https://bugzilla.redhat.com"
	bugzillaKeyFile  = "./secrets/bugzillaKey"
)

func newPriority(priority string) api.Priority {
	switch priority {
	case "low":
		return api.PriorityP5
	case "medium":
		return api.PriorityP4
	case "high":
		return api.PriorityP2
	case "urgent":
		return api.PriorityP1
	case "unspecified":
		return api.PriorityP1
	}
	logrus.Warnf("Unknown Bugzilla priority: %s", priority)
	return api.PriorityP1
}

func newAssignee(assignee string, team []config.TeamMember) string {
	for _, member := range team {
		for _, bugzillaEmail := range member.Bugzilla {
			if bugzillaEmail == assignee {
				return member.ID
			}
		}
	}
	idx := strings.Index(assignee, "@")
	if idx == -1 {
		return assignee
	}
	return assignee[:idx]
}

func newBugzillaClient() (bugzilla.Client, error) {
	apiKey, err := config.LoadSecret(bugzillaKeyFile)
	if err != nil {
		return nil, err
	}
	apiKeyFunc := func() []byte {
		return []byte(apiKey)
	}

	client := bugzilla.NewClient(apiKeyFunc, bugzillaEndpoint)
	if err := client.SetAuthMethod(bugzilla.AuthBearer); err != nil {
		return nil, err
	}
	return client, nil
}

func convertBug(bug *bugzilla.Bug, team []config.TeamMember, bugzillaClient bugzilla.Client) (*api.Task, error) {
	task := &api.Task{
		ID:      fmt.Sprintf("rhbz:%d", bug.ID),
		URL:     fmt.Sprintf("%s/show_bug.cgi?id=%d", bugzillaEndpoint, bug.ID),
		Summary: bug.Summary,
		Labels: []api.KeyValue{
			{Key: "_source", Value: "rhbz"},
			{Key: "type", Value: "Bug"},
			{Key: "priority", Value: newPriority(bug.Priority).String()},
			{Key: "status", Value: bug.Status},
			{Key: "assignee", Value: newAssignee(bug.AssignedTo, team)},
		},
	}

	if bug.Severity == "unspecified" || bug.Priority == "unspecified" {
		task.Labels.Add("flag", "untriaged")
	}

	for _, flag := range bug.Flags {
		if flag.Name == "blocker" && flag.Status == "+" {
			task.Labels.Add("flag", "blocker")
		}
		if flag.Name == "blocker" && flag.Status == "?" {
			task.Labels.Add("flag", "untriaged")
		}
		if flag.Name == "needinfo" && newAssignee(flag.Requestee, team) == team[0].ID {
			task.Labels.Add("flag", "needs-info")
		}
	}

	if len(bug.TargetRelease) > 0 && bug.TargetRelease[0] != "---" {
		task.Labels.Add("version", bug.TargetRelease[0])
	}

	for _, dep := range bug.DependsOn {
		bug, err := bugzillaClient.GetBug(dep)
		if err != nil {
			return nil, fmt.Errorf("failed to get bug %d: %w", dep, err)
		}
		if bug.Status == "VERIFIED" || bug.Status == "CLOSED" {
			continue
		}
		task.Labels.Add("blocked-by", fmt.Sprintf("rhbz:%d", dep))
	}

	return task, nil
}

type TaskSource struct {
}

func NewTaskSource() TaskSource {
	return TaskSource{}
}

func (TaskSource) LoadTasks(cfg *config.Config) ([]*api.Task, error) {
	if cfg.BugzillaQuery.Values().Encode() == "" {
		logrus.Debug("No Bugzilla query is configured.")
		return nil, nil
	}

	client, err := newBugzillaClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create bugzilla client: %w", err)
	}

	bugzillaQuery := cfg.BugzillaQuery
	bugzillaQuery.IncludeFields = []string{"id", "summary", "status", "severity", "priority", "assigned_to", "target_release", "depends_on", "flags"}

	bugs, err := client.Search(bugzillaQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search bugs: %w", err)
	}

	var tasks []*api.Task
	for _, bug := range bugs {
		task, err := convertBug(bug, cfg.Team, client)
		if err != nil {
			return nil, fmt.Errorf("failed to convert bug %d: %w", bug.ID, err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
