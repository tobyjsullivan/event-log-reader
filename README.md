# Log Query Service

An internal service to provide read access to Logs. A Log is a labeled series
of Events. A given Log is identified by a UUID. When complete, this service
should provide endpoints to:
- request all Events in a Log's history, and
- poll or subscribe for new events in a logs history

## Running with Docker Compose

```sh
cp env/sample/* env/
# Edit the env/*.env files as necessary to configure IAM

docker-compose up
```

### Connecting to the DB

You can launch `psql` if you want to play around in the DB.

```sh
docker-compose run db psql -h db -U postgres
```

## API

### GET /logs/{logId}/events

Returns a list of events as

Parameters
- `after` (optional) Only return events after this event-id

Example:

`GET /logs/{logId}/events?after={eventId}`