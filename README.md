# URL Shortener

Небольшой сервис сокращения ссылок на Go.

Что умеет:
- создавать короткие ссылки
- редиректить по короткому алиасу
- собирать аналитику по переходам
- хранить данные в PostgreSQL
- использовать Redis-кеш для чтения коротких ссылок

Инфраструктурный слой построен на `github.com/wb-go/wbf`:
- `wbf/config`
- `wbf/dbpg`
- `wbf/redis`
- `wbf/ginext`
- `wbf/zlog`

## API

`POST /shorten`

Создание короткой ссылки.

Пример запроса:
```json
{
  "url": "https://example.com/page",
  "alias": "example"
}
```

Пример ответа:
```json
{
  "status": "ok",
  "alias": "example",
  "short_url": "http://localhost:8080/s/example"
}
```

`GET /s/{short_url}`

Редирект на исходный URL.

`GET /analytics/{short_url}?group_by=raw|day|month|user_agent`

Получение аналитики по переходам.

Поддерживаемые режимы:
- `raw` — сырые события переходов
- `day` — агрегация по дням
- `month` — агрегация по месяцам
- `user_agent` — агрегация по User-Agent

## Локальный запуск

Нужно:
- Go `1.25.3`
- PostgreSQL
- Redis

Конфиг лежит в [config/local.yaml](/home/usti/projects/go/url-shortener/config/local.yaml).

Применить миграции:
```bash
make migrate
```

Запустить сервис:
```bash
make run
```

Запустить тесты:
```bash
make test
```

## Docker Compose

В проекте есть:
- [Dockerfile](/home/usti/projects/go/url-shortener/Dockerfile)
- [docker-compose.yml](/home/usti/projects/go/url-shortener/docker-compose.yml)

Compose поднимает:
- `postgres`
- `redis`
- `migrator`
- `app`
- `frontend`

Запуск:
```bash
docker compose up --build
```

После старта:
- фронтенд будет доступен на `http://localhost:8080`
- API будет доступно на `http://localhost:8082`

## Структура

- [cmd/url-shortener/main.go](/home/usti/projects/go/url-shortener/cmd/url-shortener/main.go) — HTTP-сервис
- [cmd/migrator/migrator.go](/home/usti/projects/go/url-shortener/cmd/migrator/migrator.go) — запуск миграций
- [internal/storage/postgres/postgres.go](/home/usti/projects/go/url-shortener/internal/storage/postgres/postgres.go) — работа с Postgres и Redis-кешем
- [internal/http-server/handlers/url/save/save.go](/home/usti/projects/go/url-shortener/internal/http-server/handlers/url/save/save.go) — создание короткой ссылки
- [internal/http-server/handlers/redirect/redirect.go](/home/usti/projects/go/url-shortener/internal/http-server/handlers/redirect/redirect.go) — редирект
- [internal/http-server/handlers/analytics/analytics.go](/home/usti/projects/go/url-shortener/internal/http-server/handlers/analytics/analytics.go) — аналитика
