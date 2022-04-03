package config

import (
	"io/ioutil"

	"github.com/dmage/gypd/api"
	"github.com/eparis/bugzilla"
	"sigs.k8s.io/yaml"
)

type TeamMember struct {
	ID       string   `json:"id"`
	Bugzilla []string `json:"bugzilla"`
	Jira     []string `json:"jira"`
}

type ScoreRule struct {
	Key   string
	Value string
	Score int
}

func (rule ScoreRule) Match(labels []api.KeyValue) bool {
	for _, label := range labels {
		if label.Key == rule.Key && label.Value == rule.Value {
			return true
		}
	}
	return false
}

type Config struct {
	BugzillaQuery bugzilla.Query `json:"bugzillaQuery"`
	JiraQuery     string         `json:"jiraQuery"`
	Team          []TeamMember   `json:"team"`
	ScoreRules    []ScoreRule    `json:"scoreRules"`
}

func LoadConfig() (*Config, error) {
	buf, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		return nil, err
	}
	var config Config
	if err := yaml.Unmarshal(buf, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
