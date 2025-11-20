# Link Checker Service

Сервис для проверки доступности ссылок и генерации PDF отчётов с результатами.

## Запуск 

```bash
git clone https://github.com/EugeneKrivoshein/14_11_2025_linkChecker.git
cd 14_11_2025_linkChecker
go run ./cmd/main.go
```

В сервисе использовались такие практики как:

- **Graceful shutdown** - остановка сервиса с сохранением состояния и завершением текущих задач.
- **Worker pool** - `Manager` с ограниченным числом воркеров для асинхронной обработки ссылок.
- **Concurrency safe** - использование `sync.Mutex` для защиты доступа к состоянию задач.
- **Dependency injection** - HTTP-обработчики принимают store и менеджер через конструктор.
- библиотека `gofpdf` для генерации отчетов по ссылкам.

## REST API

**POST**

Пример тела запроса:

```json
{
    "links": ["google.com", "github.com"]
}
```
Пример ответа:
```json
{
    "links": {
        "github.com": "available",
        "google.com": "available"
    },
    "links_num": 1
}
```
Порядок ссылок в ответе не гарантирован и может быть рандомным.


**POST**

Пример тела запроса:
```json
{
    "links_list": [1, 2]
}
```
Пример ответа:

```bash
Файл PDF с проверенными ссылками и их статусами.
```

Пример запроса через curl с сохранением PDF в файл:

```bash
curl -X POST http://localhost:8080 \
     -H "Content-Type: application/json" \
     -d '{"links_list": [1,2]}' \

     --output report.pdf
