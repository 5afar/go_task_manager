# Go Task Manager


Небольшой сервис управления задачами на Go с поддержкой Postgres (хранение) и Redis (кеш).

Ключевые возможности
- CRUD для задач (title, description, status)
- Слой `store` для Postgres (pgx)
- Слой `service` с валидацией бизнес-логики
- Простой Redis-клиент в `internal/cache` для кеширования
- Миграции в папке `migrations/`
- Unit-тесты, sqlmock-тесты и интеграционные тесты через `dockertest`

Требования
- Go 1.18+
- Docker (для локальной разработки и интеграционных тестов)

Быстрый старт

1) Клонируйте репозиторий и установите зависимости:

```bash
git clone <your-repo-url>
cd pet
go mod download
```

2) Запустите локальный Redis:

```bash
./scripts/start_redis.sh
```

Скрипт поднимет контейнер Redis с паролем, см. `.env.example`.

3) Примените миграции к вашей базе Postgres:

```bash
go run ./cmd/migrate -dsn "postgresql://user:pass@host:5432/dbname?sslmode=disable"
```

В разработке можно включить автоматическое применение миграций при старте сервера, установив `AUTO_MIGRATE=true` и `POSTGRES_DSN`.

4) Запустите сервер:

```bash
export REDIS_PASSWORD=your_redis_password
# export POSTGRES_DSN=...
go run ./cmd/server
```

API
- HTTP endpoints реализованы в `internal/api`.
- Примеры:
  - `GET /health` — проверка статуса
  - `POST /tasks` — создать задачу
  - `GET /tasks/:id` — получить задачу

Тесты
- Запуск всех тестов:

```bash
go test ./... -v
```

- Интеграционные тесты используют Docker (dockertest). Убедитесь, что Docker запущен.

Структура проекта
- `cmd/` — бинарники (`server`, `migrate`)
- `internal/cache` — Redis клиент
- `internal/store` — репозиторий и реализация Postgres
- `internal/service` — бизнес-логика
- `internal/api` — HTTP handlers
- `migrations/` — SQL-миграции

Run with Docker (Compose)
-------------------------

Для быстрого запуска приложения вместе с Postgres и Redis используйте `docker-compose`.

1. Создайте `.env` в корне с паролем для Redis (пример берётся из `.env.example`):

```bash
echo "REDIS_PASSWORD=your_redis_password" > .env
```

2. Построить и поднять сервисы:

```bash
docker compose up --build -d
```

3. Остановить и удалить контейнеры:

```bash
docker compose down
```

Приложение будет доступно на порту `8080` (http://localhost:8080). По умолчанию `AUTO_MIGRATE=1` в `docker-compose.yml`, поэтому миграции применяются автоматически при старте.

Если вы хотите запускать только базу данных и кеш без приложения:

```bash
docker compose up -d postgres redis
```

Важно: `docker-compose.yml` использует переменную `REDIS_PASSWORD` из файла `.env`
