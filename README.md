# Redis example
Код реазлизует простейший веб сервис отдающий информацию о пользователе по его id.
```sh
❯ http http://127.0.0.1:8080/user?id=1
HTTP/1.1 200 OK
Content-Length: 37
Content-Type: application/json
Date: Thu, 12 Dec 2024 05:08:45 GMT

{
    "age": 30,
    "id": 1,
    "name": "Test User"
}
```

Данные о пользователях хранятся в PostgreSQL. Для ускорения ответа используется Redis.

## Запуск
В папке `deployment` находится `docker-compose.yml` файл, который запускает все необходимые зависимости:
- `PostgreSQL` - основное хранилище данных
- `Redis` - кэш
- `OpenTelemetry Collector` - сборщик трейсов
- `Jaeger` - визуализация трейсов

`go run *.go` - запуск сервиса

После запуска сервиса следует сделать к нему ряд запросов. В интерфейсе [Jaeger](http://127.0.0.1:16686/) будет видно, что запросы которые идут подряд меньше чем за десять секунд, используют кэш и отрабатывают быстрее.