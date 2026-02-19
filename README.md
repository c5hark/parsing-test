Lenta Scraper

Парсер каталога https://lenta.com
 с обходом anti-bot защиты.

🚀 Возможности

1. Обход защиты Qrator
2. uTLS Chrome fingerprint
3. HTTP/2
4. Поддержка прокси
5. Пагинация каталога
6. Экспорт в CSV


🧠 Архитектура

1. Playwright прогревает сессию
2. Получаем anti-bot cookies
3. Извлекаем Utk_SessionToken
4. Выполняем API-запросы через кастомный HTTP-клиент с uTLS
5. Экспортируем данные

⚙️ Установка
go mod tidy

Установить Playwright:

go install github.com/playwright-community/playwright-go/cmd/playwright@latest
playwright install

▶ Запуск
go run main.go

С прокси:

go run main.go -proxy=http://QQsub95g:YXR8FfW7@85.142.7.134:63284
