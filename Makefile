PHONY: start
start:
	docker compose up --build

PHONY: stop
stop:
	docker compose down

PHONY: reset
reset:
	docker compose down -v
	docker compose up --build