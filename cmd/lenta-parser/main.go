package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/playwright-community/playwright-go"
	"testJob/internal/lenta"
)

// main — точка входа.
// 1. Инициализирует конфиг и HTTP-клиент с uTLS fingerprint.
// 2. Прогревает сессию через Playwright для получения anti-bot cookies.
// 3. Извлекает session token.
// 4. Итерируется по категориям и пагинирует API.
// 5. Экспортирует результат в CSV.
func main() {
	proxy := flag.String("proxy", "", "URL прокси (пример: http://user:pass@ip:port)")
	output := flag.String("output", "products.csv", "Путь к файлу выгрузки (csv или json)")
	flag.Parse()

	// Конфигурация клиента.
	// DeviceID и UserSessionID эмулируют браузерную сессию.
	// Без них API может возвращать 401/403.
	cfg := &lenta.Config{
		ProxyURL:      *proxy,
		Domain:        "lenta.com",
		DeviceID:      uuid.New().String(),
		UserSessionID: uuid.New().String(),
	}

	// Создаём HTTP-клиент с кастомным транспортом (uTLS + HTTP/2).
	// Это необходимо для эмуляции TLS fingerprint браузера.

	client, err := lenta.NewClient(cfg)
	if err != nil {
		log.Fatal("Ошибка создания клиента:", err)
	}

	log.Println("Прогрев сессии через playwright-go...")

	// Запускаем headless браузер для прохождения anti-bot (Qrator).
	// После прогрева получаем валидные cookies.
	pw, err := playwright.Run()
	if err != nil {
		log.Fatal("Ошибка запуска playwright:", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--no-sandbox",
			"--disable-infobars",
			"--window-size=1920,1080",
			"--disable-gpu",
		},
	})
	if err != nil {
		log.Fatal("Ошибка запуска браузера:", err)
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"),
		Viewport: &playwright.Size{
			Width:  1920,
			Height: 1080,
		},
		Locale: playwright.String("ru-RU"),
	})
	if err != nil {
		log.Fatal("Ошибка создания контекста:", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		log.Fatal("Ошибка создания страницы:", err)
	}

	// Переходим на главную страницу,
	// чтобы инициировать anti-bot проверку и получить первичные cookies.

	_, err = page.Goto("https://lenta.com/", playwright.PageGotoOptions{
		Timeout:   playwright.Float(60000),
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		log.Printf("Ошибка перехода на главную: %v", err)
	}
	time.Sleep(6 * time.Second)

	// Переход в категорию нужен для полной инициализации frontend-сессии.
	// Некоторые токены появляются только после загрузки каталога.

	_, err = page.Goto("https://lenta.com/catalog/moloko-128/", playwright.PageGotoOptions{
		Timeout:   playwright.Float(60000),
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		log.Printf("Ошибка перехода в категорию: %v", err)
	}
	time.Sleep(8 * time.Second)

	// Ждём появления карточек товаров
	err = page.Locator(`.card-name_content`).First().WaitFor(
		playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(30000),
		},
	)

	if err != nil {
		log.Printf("Карточки товаров не появились за 30 сек: %v", err)
	}

	// Эмуляция пользовательского поведения.
	// Скролл помогает завершить anti-bot challenge.

	_, err = page.Evaluate(`() => { window.scrollBy(0, document.body.scrollHeight / 2); }`, nil)
	time.Sleep(4 * time.Second)

	_, err = page.Evaluate(`() => { window.scrollBy(0, document.body.scrollHeight); }`, nil)
	time.Sleep(4 * time.Second)

	// Получаем cookies из браузера.
	// Они будут перенесены в http.Client для API-запросов.

	cookies, err := context.Cookies("https://lenta.com")
	if err != nil {
		log.Fatal("Ошибка получения куки:", err)
	}

	var httpCookies []*http.Cookie
	for _, c := range cookies {
		sameSite := http.SameSiteLaxMode

		if c.SameSite != nil {
			switch c.SameSite {
			case playwright.SameSiteAttributeStrict:
				sameSite = http.SameSiteStrictMode
			case playwright.SameSiteAttributeLax:
				sameSite = http.SameSiteLaxMode
			case playwright.SameSiteAttributeNone:
				sameSite = http.SameSiteNoneMode
			}
		}

		httpCookies = append(httpCookies, &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  time.Unix(int64(c.Expires), 0),
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
			SameSite: sameSite,
		})
	}

	// Переносим cookies в cookiejar HTTP-клиента.
	// После этого API-запросы будут проходить как из браузера.

	client.SetCookies(httpCookies)

	// Логируем важные куки
	for _, ck := range httpCookies {
		name := ck.Name
		if strings.Contains(name, "qrator") || name == "Utk_SessionToken" || name == "UserSessionId" {
			val := ck.Value
			if len(val) > 16 {
				val = val[:16] + "..."
			}
			log.Printf("PW Куки: %-20s = %s", name, val)
		}
	}

	// Извлекаем Utk_SessionToken.
	// Он обязателен для заголовка `sessiontoken`.

	sessionToken, err := client.ExtractSessionToken()
	if err == nil && sessionToken != "" {
		cfg.SessionToken = sessionToken
		log.Printf("Utk_SessionToken установлен: %s...", sessionToken[:16])
	} else {
		log.Println("[WARN] Utk_SessionToken НЕ найден — высокая вероятность 403/401")
	}

	categories := []struct {
		ID   int
		Name string
		Slug string
	}{
		{128, "Молочная продукция", "moloko-128"},
		// {129, "Хлеб и выпечка", "hleb-vypechka-129"},
	}

	var allProducts []lenta.ProductExport

	fmt.Println("Товар | Цена | Ссылка")

	// Итерация по категориям.
	// Для каждой категории выполняется пагинация через offset/limit.

	for _, cat := range categories {
		offset := 0
		limit := 40

		for {
			data, err := lenta.FetchCategory(client, cat.ID, offset, limit)
			if err != nil {
				log.Printf("Ошибка категории %d (%s, offset %d): %v", cat.ID, cat.Name, offset, err)
				break
			}

			for _, item := range data.Items {
				link := "https://lenta.com/p/" + item.Slug
				price := float64(item.Prices.Price) / 100

				allProducts = append(allProducts, lenta.ProductExport{
					Name:  item.Name,
					Price: price,
					URL:   link,
				})

				fmt.Printf("%s | %.2f ₽ | %s\n", item.Name, price, link)
			}

			if len(data.Items) < limit {
				break
			}

			offset += limit
			time.Sleep(2500*time.Millisecond + time.Duration(rand.Intn(3000))*time.Millisecond)
		}
	}

	if len(allProducts) > 0 && *output != "" {
		if err := lenta.ExportToCSV(allProducts, *output); err != nil {
			log.Printf("Ошибка экспорта: %v", err)
		} else {
			log.Printf("Выгружено %d товаров → %s", len(allProducts), *output)
		}
	} else {
		log.Println("Товары не собраны — проверьте куки, прокси, fingerprint")
	}
}
