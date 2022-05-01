// vim: set sw=2 et:
import './App.css';
import React, { useEffect, useState } from 'react';

import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';

import Badge from 'react-bootstrap/Badge';
import Button from 'react-bootstrap/Button';
import Card from 'react-bootstrap/Card';
import Dropdown from 'react-bootstrap/Dropdown';

function reload() {
  window.location.reload();
}

function handleErrors(response) {
  if (!response.ok) {
    throw Error(response.statusText);
  }
  return response;
}

function addMarker(task, marker, until) {
  console.log("Adding marker " + marker + " to " + task.id);
  return fetch('/api/tasks/' + encodeURIComponent(task.id) + '/markers', {
    method: 'POST',
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      marker: marker,
      until: until,
    }),
  })
    .then(handleErrors)
    .then(() => reload());
}

function createGoal(id) {
  console.log("Creating goal " + id);
  return fetch('/api/goals', {
    method: 'POST',
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      id: id,
    }),
  })
    .then(handleErrors)
    .then(() => reload());
}

function setTaskParent(a, parentID, reload) {
  console.log("Setting parent of " + a.id + " to " + parentID);
  return fetch('/api/tasks/' + encodeURIComponent(a.id) + '/parent', {
    method: 'POST',
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      id: parentID,
    }),
  })
    .then(handleErrors)
    .then(() => reload());
}


function keyValues(task, key) {
  return task.labels.reduce((result, label) => {
    const prefix = key + ': ';
    if (label.startsWith(prefix)) {
      return result.concat(label.substring(prefix.length));
    }
    return result;
  }, []);
}

function labelVariant(label) {
  return {
    'assignee: NONE': 'info',
    'flag: blocked': 'dark',
    'flag: blocker': 'warning',
    'flag: delegated': 'dark',
    'flag: needs-info': 'danger',
    'flag: needs-stories': 'danger',
    'flag: untriaged': 'danger',
    'priority: P1': 'warning',
    'priority: P2': 'info',
    'status: MODIFIED': 'warning',
    'status: ON_DEV': 'info',
    'status: POST': 'warning',
  }[label] || 'secondary';
}

function statusVariant(task) {
  const taskStatus = keyValues(task, 'status').join(', ');
  if (taskStatus === 'NEW') {
    return 'outline-primary';
  } else if (taskStatus === 'ASSIGNED' || taskStatus === 'ON_DEV' || taskStatus === 'POST') {
    if (task.labels.includes('flag: delegated')) {
      return 'outline-info';
    }
    return 'primary';
  } else if (taskStatus === 'ON_QA' || taskStatus === 'VERIFIED' || taskStatus === 'CLOSED') {
    return 'success';
  } if (task.labels.includes('_source: goal')) {
    return 'outline-secondary';
  }
  return 'secondary';
}

function cardClassName(task) {
  for (let label of task.labels) {
    if (labelVariant(label) === 'danger') {
      return 'needs-attention';
    }
  }
  for (let label of task.labels) {
    if (labelVariant(label) === 'dark') {
      return 'blocked';
    }
  }
  if (task.labels.includes('_source: goal')) {
    return 'goal';
  }
  if (task.labels.includes('type: Epic')) {
    return 'epic';
  }
}

function stringifyLabels(tasks) {
  for (let task of tasks) {
    task.labels = task.labels.map(label => label.key + ': ' + label.value);
  }
  return tasks;
}

function getGoals(tasks) {
  const goals = [];
  for (let task of tasks) {
    if (task.labels.includes('_source: goal')) {
      goals.push(task);
    }
  }
  return goals;
}

function hideChildren(tasks) {
  let m = {};
  for (let task of tasks) {
    m[task.id] = task;
    task.subtasks = [];
  }
  for (let task of tasks) {
    let p = keyValues(task, 'parent').join(', ');
    if (p !== '' && Object.prototype.hasOwnProperty.call(m, p)) {
      task.hidden = true;
      m[p].subtasks.push(task);
    }
  }
}

const BadgeToggle = React.forwardRef(({ children, onClick, className }, ref) => (
  <span
    role="button"
    className={"badge bg-secondary " + className}
    ref={ref}
    onClick={(e) => {
      e.preventDefault();
      onClick(e);
    }}
  >
    {children}
  </span>
));

function FormCreateGoal({ className, onClose }) {
  return (
    <Card className={className}>
      <Card.Header>Create Goal</Card.Header>
      <Card.Body>
        <form onSubmit={(e) => {
          e.preventDefault();
          createGoal(e.target.id.value);
          onClose();
        }}>
          <div className="form-group">
            <label htmlFor="id">Goal ID</label>
            <input type="text" className="form-control" id="id" placeholder="Goal ID" required />
          </div>
          <button type="submit" className="mt-2 btn btn-primary">Create</button>
        </form>
      </Card.Body>
    </Card>
  );
}

function TaskBody({ task, goals, reload }) {
  return (
    <div className="task">
      <div>
        <Button variant={statusVariant(task)} size="sm" href={task.url} disabled={!task.url} className={!task.url ? "disabled" : ""} target="_blank">{task.id}</Button>
      </div>
      <div className="pb-1">
        <span>{task.summary}</span><br />
        {task.labels.filter(flag => !flag.startsWith('_')).map(flag => (
          <><Badge bg={labelVariant(flag)} key={flag}>{flag}</Badge>{' '}</>
        ))}
        <Dropdown as="span">
          <Dropdown.Toggle as={BadgeToggle} variant="secondary">
            ⋮
          </Dropdown.Toggle>
          <Dropdown.Menu>
            <Dropdown.Item onClick={() => addMarker(task, "blocked", "+12h")}>Mark as blocked for 12h</Dropdown.Item>
            <Dropdown.Item onClick={() => addMarker(task, "important", "+168h")}>Mark as important for 7 days</Dropdown.Item>
            <Dropdown.Item onClick={() => addMarker(task, "later", "+12h")}>Mark as later for 12h</Dropdown.Item>
            <Dropdown.Item onClick={() => addMarker(task, "later", "+168h")}>Mark as later for 7 days</Dropdown.Item>
            <Dropdown.Divider />
            {goals.map(goal => (
              <Dropdown.Item onClick={() => setTaskParent(task, goal.id, reload)}>Add to the goal {goal.summary}</Dropdown.Item>
            ))}
          </Dropdown.Menu>
        </Dropdown>
      </div>
    </div>
  );
}

function Task({ task, className, collapseTasks, goals, reload }) {
  return (
    <Card className={(className ? className + " " : "") + cardClassName(task)}>
      <Card.Body>
        <TaskBody task={task} goals={goals} reload={reload}  />
        {collapseTasks ? (
          task.subtasks.length === 0 ? null : <div>{task.subtasks.length} subtasks</div>
        ) : (
          task.subtasks.map(t => <Task task={t} className="mb-1 ms-4" collapseTasks={collapseTasks} goals={goals} reload={reload} />)
        )}
      </Card.Body>
    </Card>
  );
}

function App() {
  const [error, setError] = useState(null);
  const [tasks, setTasks] = useState(null);
  const [loading, setLoading] = useState(true);
  const [goals, setGoals] = useState(null);
  const [showAddGoal, setShowAddGoal] = useState(false);
  const [collapseTasks, setCollapseTasks] = useState(false);

  const loadTasks = () => {
    setLoading(true);
    fetch(
      '/api/tasks', {
      headers: {
        'Accept': 'application/json',
      },
    })
      .then(response => response.json())
      .then(data => {
        setLoading(false);
        data = stringifyLabels(data);
        setGoals(getGoals(data));
        hideChildren(data);
        setTasks(data);
      })
      .catch(error => {
        setLoading(false);
        setError(error);
      })
  }

  useEffect(loadTasks, []);

  if (error) {
    return (
      <Container>
        <Row>
          <Col>
            <Card className="mt-4">
              <Card.Body>
                <Card.Title>Error loading tasks</Card.Title>
                <Card.Text>
                  {error.message}
                </Card.Text>
              </Card.Body>
            </Card>
          </Col>
        </Row>
      </Container>
    );
  }

  return (
    <Container>
      {loading ? <div className="status-bar">Loading...</div> : []}
      {tasks === null ? [] : (
        <>
          <Button
            className="mt-4"
            variant="outline-primary"
            onClick={() => setShowAddGoal(!showAddGoal)}
          >
            Add Goal
          </Button>
          <Button
            className="mt-4 ms-1"
            variant="outline-primary"
            onClick={() => setCollapseTasks(!collapseTasks)}
          >
            {collapseTasks ? "Expand Tasks" : "Collapse Tasks"}
          </Button>
          {showAddGoal && (
            <FormCreateGoal
              className="mt-2"
              onClose={() => setShowAddGoal(false)}
            />
          )}
          {tasks.filter(t => !t.hidden).map(task => (
            <Row className="mt-3 mb-3" key={task.id}>
              <Col>
                <Task task={task} collapseTasks={collapseTasks} goals={goals} reload={loadTasks} />
              </Col>
            </Row>
          ))}
        </>
      )}
    </Container>
  );
}

export default App;
