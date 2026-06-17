ENV_FILE ?= .env.production
COMPOSE = docker compose --env-file $(ENV_FILE) -f docker-compose.production.yml

.PHONY: deploy up down restart ps logs check-health pull

deploy:
	ENV_FILE=$(ENV_FILE) ./scripts/deploy-production.sh

pull:
	$(COMPOSE) pull

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

restart:
	$(COMPOSE) restart

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs --tail=100 -f

check-health:
	@$(COMPOSE) exec -T app wget -qO- http://localhost:8080/healthz && echo " /healthz OK" || echo " /healthz FAILED"
	@$(COMPOSE) exec -T app wget -qO- http://localhost:8080/readyz  && echo " /readyz  OK" || echo " /readyz  FAILED"
