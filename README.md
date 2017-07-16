# Log Query Service

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

