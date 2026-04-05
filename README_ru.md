# Fragment API Go Library

[![Go 1.21+](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/Darkildo/fragment-api-go)

**Go-клиент для Fragment.com API -- Telegram Stars, Premium-подписки и TON-переводы.**

Go-порт библиотеки [fragment-api-py](https://github.com/S1qwy/fragment-api-py) (Python v3.2.0).

[README in English](README.md)

---

## Содержание

- [Возможности](#возможности)
- [Структура проекта](#структура-проекта)
- [Установка](#установка)
- [Быстрый старт](#быстрый-старт)
- [Настройка](#настройка)
- [API-справочник](#api-справочник)
- [Версии кошельков](#версии-кошельков)
- [Обработка ошибок](#обработка-ошибок)
- [Модели данных](#модели-данных)
- [Статус разработки](#статус-разработки)
- [Лицензия](#лицензия)

---

## Возможности

- **Telegram Stars** -- отправка Stars любому пользователю Telegram
- **Premium-подарки** -- подарок подписки Telegram Premium на 3/6/12 месяцев
- **Пополнение TON Ads** -- пополнение рекламных аккаунтов TON Ads
- **Прямые TON-переводы** -- отправка TON на любой адрес с опциональным комментарием
- **Управление кошельком** -- запрос баланса, поддержка нескольких версий (V3R1, V3R2, V4R2, V5R1/W5)
- **Видимость отправителя** -- анонимные или видимые платежи
- **Автоматические повторы** -- экспоненциальный backoff при сетевых ошибках
- **Типизированные ошибки** -- отдельный тип ошибки для каждого сценария
- **Поддержка context.Context** -- отмена и таймауты для всех операций

---

## Структура проекта

```
fragment-api-go/
├── fragment.go          # Корневой пакет -- версия, документация
├── go.mod               # Определение Go-модуля
│
├── client/
│   └── client.go        # Высокоуровневый клиент FragmentAPI (основная точка входа)
│
├── core/
│   └── core.go          # Низкоуровневый HTTP-клиент для Fragment.com API
│
├── wallet/
│   └── wallet.go        # Управление TON-кошельком (баланс, транзакции, переводы)
│
├── models/
│   └── models.go        # Структуры данных: UserInfo, PurchaseResult и др.
│
├── errors/
│   └── errors.go        # Иерархия типов ошибок
│
├── utils/
│   └── utils.go         # Утилиты: парсинг cookies, валидация, конвертация TON
│
├── example/
│   └── main.go          # Пример использования
│
├── README.md            # Документация (English)
└── README_ru.md         # Документация (Русский)
```

### Описание пакетов

| Пакет | Описание |
|-------|----------|
| `client` | **Основная точка входа.** Структура `FragmentAPI` с методами `BuyStars`, `GiftPremium`, `TopupTON`, `TransferTON`, `GetWalletBalance` |
| `core` | HTTP-клиент: управление сессией, cookies, повторы запросов, парсинг JSON |
| `wallet` | Интеграция с TON-блокчейном: инициализация кошелька, баланс, транзакции (скелет -- требуется TON SDK) |
| `models` | Типы данных: `UserInfo`, `PurchaseResult`, `TransferResult`, `WalletBalance`, `TransactionMessage` |
| `errors` | Типизированные ошибки: `AuthenticationError`, `UserNotFoundError`, `InsufficientBalanceError` и др. |
| `utils` | Парсинг cookies, валидация username, валидация сумм, конвертация TON/nanoton |

---

## Установка

```bash
go get github.com/Darkildo/fragment-api-go
```

---

## Быстрый старт

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/Darkildo/fragment-api-go/client"
)

func main() {
    api, err := client.New(client.Config{
        Cookies:        "stel_ssid=...; stel_token=...; stel_dt=...; stel_ton_token=...",
        HashValue:      "ваш_hash_из_network_tab",
        WalletMnemonic: "word1 word2 ... word24",
        WalletAPIKey:   "ваш_tonapi_ключ",
        WalletVersion:  "V4R2",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer api.Close()

    ctx := context.Background()

    // Поиск пользователя
    user, err := api.GetRecipientStars(ctx, "jane_doe")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Пользователь: %s\n", user.Name)

    // Отправить 100 Stars (анонимно)
    result, err := api.BuyStars(ctx, "jane_doe", 100, false)
    if err != nil {
        log.Fatal(err)
    }
    if result.Success {
        fmt.Printf("TX: %s\n", result.TransactionHash)
    }

    // Отправить Stars с видимым отправителем
    result, err = api.BuyStars(ctx, "jane_doe", 50, true)

    // Подарить Premium (3 месяца, анонимно)
    premResult, err := api.GiftPremium(ctx, "jane_doe", 3, false)

    // Прямой перевод TON с комментарием
    transfer, err := api.TransferTON(ctx, "recipient.t.me", 0.5, "Оплата услуг")

    // Проверить баланс кошелька
    balance, err := api.GetWalletBalance(ctx)
    if err == nil {
        fmt.Printf("Баланс: %.6f TON\n", balance.BalanceTON)
    }
}
```

---

## Настройка

### 1. Извлечь cookies Fragment

1. Откройте [fragment.com](https://fragment.com) в браузере
2. Нажмите `F12` для открытия DevTools
3. Перейдите в `Application` > `Cookies` > `fragment.com`
4. Скопируйте cookies:

| Cookie | Назначение |
|--------|-----------|
| `stel_ssid` | ID сессии |
| `stel_token` | Токен аутентификации |
| `stel_dt` | Смещение часового пояса |
| `stel_ton_token` | TON-токен |

5. Объедините в одну строку:
```
stel_ssid=abc123; stel_token=xyz789; stel_dt=-180; stel_ton_token=uvw012
```

### 2. Получить Hash

1. Оставьте DevTools открытым, перейдите на вкладку `Network`
2. Обновите fragment.com
3. Найдите запросы к `fragment.com/api`
4. Скопируйте значение параметра `hash`

### 3. Подготовить TON-кошелек

Экспортируйте 24-словную мнемоническую фразу из вашего TON-кошелька (Tonkeeper, MyTonWallet, TonHub).

Версии кошельков по умолчанию:
- **Tonkeeper** -- V4R2
- **MyTonWallet** -- V4R2
- **TonHub** -- V5R1

### 4. Получить TonAPI-ключ

1. Перейдите на [tonconsole.com](https://tonconsole.com)
2. Создайте проект
3. Скопируйте API Key

### 5. Переменные окружения

```bash
export FRAGMENT_COOKIES="stel_ssid=...; stel_token=...; stel_dt=...; stel_ton_token=..."
export FRAGMENT_HASH="abc123def456..."
export WALLET_MNEMONIC="word1 word2 ... word24"
export WALLET_API_KEY="ваш_ключ"
export WALLET_VERSION="V4R2"
```

---

## API-справочник

### Методы клиента

```go
// Создание клиента
api, err := client.New(cfg client.Config) (*client.FragmentAPI, error)

// Поиск получателя
user, err := api.GetRecipientStars(ctx, username)   // -> *models.UserInfo
user, err := api.GetRecipientPremium(ctx, username)  // -> *models.UserInfo
user, err := api.GetRecipientTON(ctx, username)      // -> *models.UserInfo

// Покупки
result, err := api.BuyStars(ctx, username, quantity, showSender)     // -> *models.PurchaseResult
result, err := api.GiftPremium(ctx, username, months, showSender)    // -> *models.PurchaseResult
result, err := api.TopupTON(ctx, username, amount, showSender)       // -> *models.PurchaseResult

// Прямой перевод
transfer, err := api.TransferTON(ctx, toAddress, amountTON, memo)   // -> *models.TransferResult

// Кошелек
balance, err := api.GetWalletBalance(ctx)   // -> *models.WalletBalance
info := api.GetWalletInfo()                 // -> map[string]interface{}
```

### Параметры

| Метод | Параметр | Тип | Описание |
|-------|----------|-----|----------|
| `BuyStars` | `username` | `string` | Telegram username (5-32 символа) |
| | `quantity` | `int` | Количество звезд (1-999999) |
| | `showSender` | `bool` | Показать отправителя |
| `GiftPremium` | `months` | `int` | Длительность: 3, 6 или 12 |
| `TopupTON` | `amount` | `int` | Сумма в TON (1-999999) |
| `TransferTON` | `toAddress` | `string` | TON-адрес или `user.t.me` |
| | `amountTON` | `float64` | Сумма в TON |
| | `memo` | `string` | Комментарий к транзакции |

---

## Версии кошельков

| Версия | Название | Статус | Применение |
|--------|----------|--------|-----------|
| **V3R1** | WalletV3R1 | Legacy | Старые кошельки |
| **V3R2** | WalletV3R2 | Legacy | Старые кошельки |
| **V4R2** | WalletV4R2 | **Рекомендуется** | Максимальная совместимость |
| **V5R1** | WalletV5R1 | Новейший | Современные функции |
| **W5** | Алиас для V5R1 | Новейший | Альтернативное название |

Версия регистронезависима: `"v4r2"`, `"V4R2"`, `"V4r2"` -- все работают.

---

## Обработка ошибок

```go
import fragErrors "github.com/Darkildo/fragment-api-go/errors"
```

### Иерархия ошибок

```
FragmentAPIError (базовая)
├── AuthenticationError         — сессия истекла / невалидные credentials
├── UserNotFoundError           — пользователь не найден в Telegram
├── InvalidAmountError          — количество/сумма вне допустимого диапазона
├── InsufficientBalanceError    — недостаточный баланс кошелька
├── PaymentInitiationError      — Fragment API отклонил инициацию платежа
├── TransactionError            — ошибка исполнения транзакции в блокчейне
├── NetworkError                — ошибка HTTP-запроса
├── RateLimitError              — превышен лимит запросов
└── WalletError                 — общая ошибка кошелька
    └── InvalidWalletVersionError — неподдерживаемая версия кошелька
```

### Пример

```go
import (
    "errors"
    fragErrors "github.com/Darkildo/fragment-api-go/errors"
)

result, err := api.BuyStars(ctx, "username", 100, false)
if err != nil {
    var authErr *fragErrors.AuthenticationError
    var userErr *fragErrors.UserNotFoundError
    var balErr  *fragErrors.InsufficientBalanceError

    switch {
    case errors.As(err, &authErr):
        log.Println("Сессия истекла — обновите cookies")
    case errors.As(err, &userErr):
        log.Printf("Пользователь не найден: %s", userErr.Username)
    case errors.As(err, &balErr):
        log.Printf("Нужно %.6f TON, есть %.6f", balErr.Required, balErr.Current)
    default:
        log.Printf("Ошибка: %v", err)
    }
}
```

---

## Модели данных

### UserInfo

```go
type UserInfo struct {
    Name      string // Отображаемое имя
    Recipient string // Адрес получателя в блокчейне
    Found     bool   // Найден ли пользователь
    Avatar    string // URL аватара или base64
}
```

### PurchaseResult

```go
type PurchaseResult struct {
    Success         bool    // Успех транзакции
    TransactionHash string  // Хеш транзакции в блокчейне
    Error           string  // Сообщение об ошибке (при неудаче)
    User            *UserInfo
    BalanceChecked  bool    // Баланс был проверен
    RequiredAmount  float64 // Итоговая стоимость в TON
}
```

### TransferResult

```go
type TransferResult struct {
    Success         bool
    TransactionHash string
    FromAddress     string
    ToAddress       string
    AmountTON       float64
    BalanceBefore   float64
    Memo            string
    Error           string
}
```

### WalletBalance

```go
type WalletBalance struct {
    BalanceNano   string  // Баланс в нанотонах
    BalanceTON    float64 // Баланс в TON
    Address       string  // Адрес кошелька
    IsReady       bool    // Готовность кошелька
    WalletVersion string  // Версия контракта
}
```

---

## Статус разработки

Библиотека представляет собой **структурный скелет** (v1.0.0). HTTP-клиент (`core`), модели, ошибки, утилиты и высокоуровневый API полностью определены.

**Что требует реализации:**

Пакет `wallet` возвращает заглушки. Для полной функциональности необходимо интегрировать Go TON SDK:

- [xssnick/tonutils-go](https://github.com/xssnick/tonutils-go) -- полноценный TON SDK
- [tonkeeper/tongo](https://github.com/tonkeeper/tongo) -- TON-библиотека от Tonkeeper

Необходимые операции кошелька:
1. **`GetBalance`** -- вывести адрес из мнемоники, запросить баланс через TonAPI
2. **`SendTransaction`** -- декодировать BOC-payload, подписать и отправить транзакцию
3. **`TransferTON`** -- построить перевод с опциональным memo-cell, отправить в сеть

Каждый метод в `wallet/wallet.go` содержит подробный псевдокод и TODO-комментарии с описанием шагов реализации.

---

## Лицензия

MIT License. См. [LICENSE](LICENSE).

Основано на [fragment-api-py](https://github.com/S1qwy/fragment-api-py) от [S1qwy](https://github.com/S1qwy).
