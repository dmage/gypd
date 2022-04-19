package rh

import (
	"fmt"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/dmage/gypd/api"
	"github.com/dmage/gypd/config"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var (
	jiraEndpoint  = "https://issues.redhat.com"
	jiraTokenFile = "./secrets/jiraToken"
)

const epicLinkField = "customfield_12311140"

func newJiraClient() (*jira.Client, error) {
	jiraToken, err := config.LoadSecret(jiraTokenFile)
	if err != nil {
		return nil, err
	}

	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: jiraToken},
	)
	return jira.NewClient(
		oauth2.NewClient(oauth2.NoContext, tokenSource),
		jiraEndpoint,
	)
}

func newPriority(priority string) api.Priority {
	switch priority {
	case "Minor":
		return api.PriorityP4
	case "Normal", "Undefined":
		return api.PriorityP3
	case "Major", "Critical":
		return api.PriorityP2
	}
	logrus.Warnf("Unknown Jira priority: %s", priority)
	return api.PriorityP1
}

func newStatus(key, status string) api.Status {
	if strings.HasPrefix(key, "PROJQUAY-") {
		switch status {
		case "Triage":
			return api.StatusNew
		case "Plan", "Open":
			return api.StatusAssigned
		case "Coding In Progress":
			return api.StatusOnDev
		case "Pull Request Sent":
			return api.StatusPost
		case "Resolved":
			return api.StatusVerified
		case "Closed":
			return api.StatusClosed
		}
	} else if strings.HasPrefix(key, "IR-") {
		switch status {
		case "New", "Planning", "To Do":
			return api.StatusNew
		case "Approved":
			return api.StatusAssigned
		case "In Progress":
			return api.StatusOnDev
		case "Code Review":
			return api.StatusPost
		case "Review":
			return api.StatusOnQA
		case "Closed":
			return api.StatusClosed
		}
	}
	logrus.Warnf("Unknown Jira status: %s (%s)", status, key)
	return api.Status(status)
}

func newAssignee(assignee *jira.User, team []config.TeamMember) string {
	if assignee == nil {
		return api.AssigneeNone
	}
	for _, member := range team {
		for _, jiraLogin := range member.Jira {
			if jiraLogin == assignee.Name {
				return member.ID
			}
		}
	}
	idx := strings.Index(assignee.Name, "@")
	if idx == -1 {
		return assignee.Name
	}
	return assignee.Name[:idx]
}

func loadEpicLinks(jiraClient *jira.Client, key string) ([]string, error) {
	var links []string
	err := jiraClient.Issue.SearchPages(
		"\"Epic Link\" = "+key,
		&jira.SearchOptions{
			StartAt:    0,
			MaxResults: 50,
			Fields:     []string{"key"},
		},
		func(issue jira.Issue) error {
			links = append(links, issue.Key)
			return nil
		},
	)
	if err != nil {
		return links, fmt.Errorf("failed to load epic links for %s: %w", key, err)
	}
	return links, nil
}

func convertIssue(issue jira.Issue, team []config.TeamMember, jiraClient *jira.Client) (*api.Task, error) {
	task := &api.Task{
		ID:      fmt.Sprintf("rh:%s", issue.Key),
		URL:     fmt.Sprintf("%s/browse/%s", jiraEndpoint, issue.Key),
		Summary: issue.Fields.Summary,
		Labels: []api.KeyValue{
			{Key: "type", Value: issue.Fields.Type.Name},
			{Key: "priority", Value: newPriority(issue.Fields.Priority.Name).String()},
			{Key: "status", Value: newStatus(issue.Key, issue.Fields.Status.Name).String()},
			{Key: "assignee", Value: newAssignee(issue.Fields.Assignee, team)},
		},
	}

	if epic, err := issue.Fields.Unknowns.String(epicLinkField); err == nil {
		task.Labels.Add("parent", fmt.Sprintf("rh:%s", epic))
	}

	if issue.Fields.Type.Name == "Epic" {
		var err error
		epicLinks, err := loadEpicLinks(jiraClient, issue.Key)
		if err != nil {
			return task, err
		}
		if len(epicLinks) == 0 {
			task.Labels.Add("flag", "needs-stories")
		}
	}

	return task, nil
}

type TaskSource struct {
}

func NewTaskSource() TaskSource {
	return TaskSource{}
}

func (TaskSource) LoadTasks(config *config.Config) ([]*api.Task, error) {
	if config.JiraQuery == "" {
		logrus.Debug("No Jira query is configured.")
		return nil, nil
	}

	jiraClient, err := newJiraClient()
	if err != nil {
		return nil, err
	}

	logrus.Debugf("Loading Jira issues.")
	var tasks []*api.Task
	err = jiraClient.Issue.SearchPages(
		config.JiraQuery,
		&jira.SearchOptions{
			StartAt:    0,
			MaxResults: 50,
			Fields:     []string{"key", "issuetype", "summary", "status", "priority", "assignee", "components", epicLinkField},
		},
		func(issue jira.Issue) error {
			task, err := convertIssue(issue, config.Team, jiraClient)
			if err != nil {
				return err
			}
			tasks = append(tasks, task)
			return nil
		},
	)
	if err != nil {
		return tasks, fmt.Errorf("failed to search jira issues: %w", err)
	}

	return tasks, nil
}
