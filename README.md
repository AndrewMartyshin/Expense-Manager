# РасходМенеджер — базовый CRD backend

Go + Gin + GORM + Postgres. Create / Read / Delete для profiles, categories, currencies, transactions.

## Запуск

```bash
docker compose up -d          # Postgres на :5432
go mod tidy                   # подтянуть зависимости
go run .                      # сервер на :8080, таблицы создаются автоматически
```

Переменные окружения: `DATABASE_URL`, `PORT`.

## Эндпоинты

| Метод | Путь | Действие |
| --- | --- | --- |
| POST / GET | `/api/profiles` | создать / список |
| GET / DELETE | `/api/profiles/:id` | получить / удалить |
| POST / GET | `/api/categories` | то же |
| GET / DELETE | `/api/categories/:id` | то же |
| POST / GET | `/api/currencies` | то же |
| GET / DELETE | `/api/currencies/:code` | то же (ключ — код валюты) |
| POST / GET | `/api/transactions` | то же (список: свежие сверху) |
| GET / DELETE | `/api/transactions/:id` | то же |

## Быстрая проверка

```bash
curl -X POST localhost:8080/api/profiles   -d '{"name":"Аня"}'
curl -X POST localhost:8080/api/currencies -d '{"code":"RUB","name":"Рубль","minor_units":2}'
curl -X POST localhost:8080/api/categories -d '{"code":"food","name":"Продукты","type":"expense"}'
curl -X POST localhost:8080/api/transactions -d '{
  "profile_id":1,"category_id":1,"currency":"RUB","type":"expense",
  "amount_minor":45050,"occurred_on":"2026-11-26","description":"Пятёрочка",
  "idempotency_key":"abc-1"}'
curl localhost:8080/api/transactions
curl -X DELETE localhost:8080/api/transactions/1
```

`amount_minor` — сумма в минимальных единицах (45050 = 450,50 ₽).
Повторный POST с тем же `idempotency_key` для того же профиля вернёт уже созданную операцию (200), а не дубль.
