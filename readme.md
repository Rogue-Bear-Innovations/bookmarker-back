# Backend for the Bookmarker app

## Functional tests
```shell
make func-test
```

## Run locally
```shell
make local
```

## Deployment

### Building Docker image
```shell
make build
```

### Deploy to Heroku
To deploy to Heroku download the Heroku CLI and log in to your project and then run:
```shell
make deploy-heroku
```
You may also need to configure Heroku PostgreSQL as well as the environment variables for the app at the Heroku website.

### See Heroku logs
```shell
make heroku-logs
```

## gRPC
```shell
make proto-generate
```
