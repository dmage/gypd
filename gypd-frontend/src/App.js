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
  if (task.labels.includes('type: Epic')) {
    return 'epic';
  }
}

function stringifyLabels(tasks) {
  console.log(tasks);
  for (let task of tasks) {
    task.labels = task.labels.map(label => label.key + ': ' + label.value);
  }
  return tasks;
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

function reload() {
  window.location.reload();
}

function addMarker(task, marker, until) {
  console.log("Adding marker " + marker + " to " + task.id);
  const url = '/api/add-marker?id=' + encodeURIComponent(task.id) + '&marker=' + encodeURIComponent(marker) + '&until=' + encodeURIComponent(until);
  return fetch(url, {headers: {'Accept': 'application/json'}})
    .then(() => reload());
}

function App() {
  const [error, setError] = useState(null);
  const [tasks, setTasks] = useState(null);

  useEffect(() => {
    fetch(
      '/api/tasks', {
      headers: {
        'Accept': 'application/json',
      },
    })
      .then(response => response.json())
      .then(data => setTasks(stringifyLabels(data)))
      .catch(error => setError(error));
  }, []);

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

  if (tasks === null) {
    return (
      <Container>
        <Row>
          <Col className="mt-4">
            Loading...
          </Col>
        </Row>
      </Container>
    );
  }

  return (
    <Container>
      {tasks.map(task => (
        <Row className="mt-3 mb-3" key={task.id}>
          <Col>
            <Card className={cardClassName(task)}>
              <Card.Body>
                <div>
                  <Button variant={statusVariant(task)} size="sm" href={task.url} target="_blank">{task.id}</Button>
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
                    </Dropdown.Menu>
                  </Dropdown>
                </div>
              </Card.Body>
            </Card>
          </Col>
        </Row>
      ))}
    </Container>
  );
}

export default App;
