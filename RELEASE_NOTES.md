# Release Notes

## 0.1.0 - Initial release

Дата: 2026-05-03

В этой версии:
- Базовый CRUD для задач (Postgres) (`internal/store`, `internal/service`).
- Redis клиент и простое кеширование (`internal/cache`).
- HTTP API для задач (`internal/api`) с эндпоинтами create/get/list/update/delete.
- Миграции в `migrations/001_create_tasks.sql` и CLI `cmd/migrate`.
- Unit-тесты, sqlmock тесты для репозитория и интеграционные тесты с `dockertest`.
- GitHub Actions: `ci.yml` (тесты + coverage), `lint.yml` (golangci-lint).
- README и CONTRIBUTING добавлены.

Известные ограничения / TODO:
- Добавить более полную валидацию в API и расширенные интеграционные сценарии.
- Настроить релизный pipeline и семантическое версионирование.

Как обновиться
- Для локального запуска: установите зависимости и выполните миграции (`go run ./cmd/migrate`).
