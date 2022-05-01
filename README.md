# Get Your Plans Done

## Secrets

### ./secrets/bugzillaKey

You can get it from [Preferences -> API Keys](https://bugzilla.redhat.com/userprefs.cgi?tab=apikey).

### ./secrets/jiraToken

You can get it from [Profile -> Personal Access Tokens](https://issues.redhat.com/secure/ViewProfile.jspa?selectedTab=com.atlassian.pats.pats-plugin:jira-user-personal-access-tokens).

## Sample config.yaml

```yaml
bugzillaQuery:
  product:
  - OpenShift Container Platform
  component:
  - ImageStreams
  - Image Registry
  status:
  - NEW
  - ASSIGNED
  - ON_DEV
  - POST
  - MODIFIED
jiraQuery: |-
  project = IR AND statusCategory != Done
team:
- id: obulatov
scoreRules:
- {key: "flag", value: "blocker", score: 500}
```

## Building and running

```console
$ make build
$ ./gypd
```
