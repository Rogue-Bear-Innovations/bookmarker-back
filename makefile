func-test:
	docker-compose -f docker-compose.functional.yml rm -svf
	docker-compose -f docker-compose.functional.yml up --build --exit-code-from test-runner

# run locally
local:
	docker-compose -f docker-compose.local.yml rm -svf
	docker-compose -f docker-compose.local.yml up --build

# build production image
build:
	docker build . -f Dockerfile -t bookmarker:latest

deploy-heroku:
	heroku container:login
	docker tag bookmarker:latest registry.heroku.com/the-ultimate-bookmarker-app/web
	docker push registry.heroku.com/the-ultimate-bookmarker-app/web
	heroku container:release web -a the-ultimate-bookmarker-app

heroku-logs:
	heroku logs --tail -a the-ultimate-bookmarker-app
