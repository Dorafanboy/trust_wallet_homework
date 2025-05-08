# Ethereum Blockchain Parser (trust_wallet_homework)

## Обзор

[Здесь будет краткое описание проекта]

## Основные возможности

- [Возможность 1]
- [Возможность 2]
- [Возможность 3]

## Технологический стек

- Go
- Docker
- Makefile
- Gin (для REST API)
- slog (для логирования)
- testify (для тестов)
- mockery (для генерации моков)
- golangci-lint (для линтинга)

## Требования

- Go (версия 1.2x или выше)
- Docker и Docker Compose (для запуска в контейнере)
- Make

## Установка и Сборка

1. Клонируйте репозиторий:
   ```bash
   git clone <URL_РЕПОЗИТОРИЯ>
   cd trust_wallet_homework
   ```

2. Соберите приложение:
   ```bash
   make build
   ```
   Или напрямую:
   ```bash
   go build -o parserapi ./cmd/parserapi/main.go
   ```

## Запуск приложения

### Конфигурация

Приложение использует конфигурационный файл `config/config.yml`. Убедитесь, что он настроен правильно перед запуском. Ключевые параметры:

- `server.port`: Порт для API сервера (например, `":8080"`).
- `ethereum.rpc_url`: URL вашего Ethereum RPC провайдера.
- `parser.polling_interval_seconds`: Интервал опроса новых блоков (в секундах).
- `parser.initial_scan_block_number`: Номер блока для начала сканирования (-1 для начала с последнего блока, 0 или положительное число для конкретного блока).

Пример `config/config.yml`:
```yaml
server:
  port: ":8080"

ethereum:
  rpc_url: "https://cloudflare-eth.com" # Замените на ваш RPC URL

parser:
  polling_interval_seconds: 15
  initial_scan_block_number: -1 # Начать с последнего блока
```

### Локальный запуск

```bash
make run
```
Или после сборки:
```bash
./parserapi
```

### Запуск с Docker

1. Соберите Docker образ:
   ```bash
   make docker-build
   ```

2. Запустите контейнер:
   ```bash
   make docker-run
   ```
   Или используя Docker Compose (предпочтительно, если есть зависимости):
   ```bash
   make infra-up
   ```
   Остановить сервисы, запущенные через Docker Compose:
   ```bash
   make infra-down
   ```

## API Эндпоинты

- `GET  /current_block`: Возвращает номер последнего обработанного блока.
- `POST /subscribe`: Подписывает новый адрес для отслеживания транзакций.
  - Тело запроса: `{"address":"0xYOUR_ADDRESS_HERE"}`
- `GET  /transactions/{address}`: Возвращает список транзакций для указанного отслеживаемого адреса.

## Тестирование

Для запуска всех тестов:
```bash
make test
```

## Линтинг

Для проверки кода линтером:
```bash
make lint
```

## Структура проекта

[Краткий обзор структуры директорий, если необходимо]

## TODO / Возможные улучшения

- [Идея 1]
- [Идея 2]

## Лицензия

[Например, MIT License] 