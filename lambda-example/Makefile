build:
	docker compose run --rm app

bash:
	docker compose run --rm app bash

deploy: build
	docker compose run --rm app cdk deploy --require-approval never

destroy:
	docker compose run --rm app cdk destroy lambda-example
