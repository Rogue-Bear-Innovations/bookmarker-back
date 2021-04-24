func-test:
	docker-compose -f docker-compose.functional.yml rm -svf
	docker-compose -f docker-compose.functional.yml up --build --exit-code-from test-runner
