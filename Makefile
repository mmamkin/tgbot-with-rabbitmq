.PHONY: run
run: compose.yml
	docker compose up --build -d

.PHONY: stop
stop: compose.yml
	docker compose down

.DEFAULT_GOAL := run
