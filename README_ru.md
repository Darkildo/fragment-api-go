# Fragment API Go

[![Go 1.21+](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/Darkildo/fragment-api-go)

**Go-клиент для Fragment.com API — Telegram Stars, Premium-подписки и TON-переводы.**

Go-порт библиотеки [fragment-api-py](https://github.com/S1qwy/fragment-api-py) (Python v3.2.0).

[README in English](README.md)

---

## Установка

```bash
go get github.com/Darkildo/fragment-api-go
```

## Быстрый старт

```go
package main

import (
    "context"
    "fmt"
    "log"

    fragment "github.com/Darkildo/fragment-api-go"
)

func main() {
    api, err := fragment.New(fragment.Config{
        Cookies:        "stel_ssid=...; stel_token=...; stel_dt=...; stel_ton_token=...",
        HashValue:      "ваш_hash_из_network_tab",
        WalletMnemonic: "word1 word2 ... word24",
        WalletAPIKey:   "ваш_tonapi_ключ",
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

    // Отправить 100 Stars анонимно
    result, err := api.BuyStars(ctx, "jane_doe", 100, false)
    if err != nil {
        log.Fatal(err)
    }
    if result.Success {
        fmt.Printf("TX: %s\n", result.TransactionHash)
    }
}
```

## Возможности

- **Telegram Stars** — отправка Stars любому пользователю Telegram
- **Premium-подарки** — подарок подписки Telegram Premium на 3/6/12 месяцев
- **Пополнение TON Ads** — пополнение рекламных аккаунтов
- **Прямые TON-переводы** — отправка TON на любой адрес с комментарием
- **Мульти-кошелек** — V3R1, V3R2, V4R2 (по умолчанию), V5R1/W5
- **Видимость отправителя** — анонимные или видимые платежи
- **Автоматические повторы** — экспоненциальный backoff при сетевых ошибках
- **Типизированные ошибки** — отдельный тип для каждого сценария
- **Поддержка context.Context** — отмена и таймауты

## Структура проекта

Библиотека следует Go best practice: один плоский пакет, файлы разделены
по функциональности. Один путь импорта — все публичные типы доступны.

```
fragment-api-go/
  go.mod
  fragment.go    — Client, New(), Config, Close, WalletBalance, WalletInfo
  types.go       — UserInfo, PurchaseResult, TransferResult, WalletBalance
  errors.go      — APIError, AuthenticationError, UserNotFoundError, ...
  recipient.go   — GetRecipientStars, GetRecipientPremium, GetRecipientTON
  stars.go       — BuyStars
  premium.go     — GiftPremium
  topup.go       — TopupTON
  transfer.go    — TransferTON
  purchase.go    — общий флоу покупки (неэкспортируемый)
  core.go        — HTTP-транспорт (неэкспортируемый)
  wallet.go      — управление TON-кошельком (неэкспортируемый)
  helpers.go     — валидация, парсинг, конвертация (неэкспортируемый)
  example/
    main.go      — пример использования
```

| Файл | Содержимое |
|------|-----------|
| `fragment.go` | Структура `Client`, конструктор `New()`, `Config`, `Close()` |
| `types.go` | `UserInfo`, `PurchaseResult`, `TransferResult`, `WalletBalance` |
| `errors.go` | `APIError` (базовый), `AuthenticationError`, `UserNotFoundError`, `InvalidAmountError`, `InsufficientBalanceError`, `PaymentInitiationError`, `TransactionError`, `NetworkError`, `RateLimitError`, `WalletError`, `InvalidWalletVersionError` |
| `recipient.go` | `GetRecipientStars`, `GetRecipientPremium`, `GetRecipientTON` |
| `stars.go` | `BuyStars` |
| `premium.go` | `GiftPremium` |
| `topup.go` | `TopupTON` |
| `transfer.go` | `TransferTON` |
| `purchase.go` | Общий флоу покупки для Stars/Premium/TopUp (неэкспортируемый) |
| `core.go` | `httpCore` — HTTP-клиент, cookies, повторы (неэкспортируемый) |
| `wallet.go` | `walletManager` — операции с TON-кошельком (неэкспортируемый) |
| `helpers.go` | Парсинг cookies, валидация, конвертация TON/nano (неэкспортируемый) |

---

## Настройка

### 1. Извлечь cookies Fragment

1. Откройте [fragment.com](https://fragment.com), нажмите `F12`
2. `Application` > `Cookies` > `fragment.com`
3. Скопируйте: `stel_ssid`, `stel_token`, `stel_dt`, `stel_ton_token`
4. Объедините: `"stel_ssid=abc; stel_token=xyz; stel_dt=-180; stel_ton_token=uvw"`

### 2. Получить Hash

1. DevTools > вкладка `Network`, обновите fragment.com
2. Найдите запрос к `fragment.com/api`
3. Скопируйте параметр `hash`

### 3. Подготовить TON-кошелек

Экспортируйте 24-словную мнемоническую фразу из кошелька.

| Приложение | Версия по умолчанию |
|------------|-------------------|
| Tonkeeper | V4R2 |
| MyTonWallet | V4R2 |
| TonHub | V5R1 |

### 4. Получить TonAPI-ключ

1. Перейдите на [tonconsole.com](https://tonconsole.com)
2. Создайте проект, скопируйте API Key

---

## API-справочник

```go
// Создание клиента (WalletVersion по умолчанию "V4R2")
api, err := fragment.New(fragment.Config{ ... })
defer api.Close()

// Поиск получателя
user, err := api.GetRecipientStars(ctx, "username")
user, err := api.GetRecipientPremium(ctx, "username")
user, err := api.GetRecipientTON(ctx, "username")

// Покупки
result, err := api.BuyStars(ctx, "username", 100, false)
result, err := api.GiftPremium(ctx, "username", 3, false)
result, err := api.TopupTON(ctx, "username", 10, false)

// Прямой перевод
transfer, err := api.TransferTON(ctx, "addr.t.me", 0.5, "memo")

// Кошелек
balance, err := api.WalletBalance(ctx)
info := api.WalletInfo()
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
| | `memo` | `string` | Комментарий ("" без комментария) |

---

## Версии кошельков

| Версия | Статус | Примечание |
|--------|--------|-----------|
| **V4R2** | **По умолчанию** | Максимальная совместимость |
| **V5R1** | Новейший | Современные функции |
| **W5** | Алиас | Маппится в V5R1 |
| **V3R2** | Legacy | Старые кошельки |
| **V3R1** | Legacy | Старые кошельки |

Регистронезависимо: `"v4r2"`, `"V4R2"`, `"V4r2"` — все работают.

---

## Обработка ошибок

Все типы ошибок встраивают `APIError` и реализуют интерфейс `error`.
Используйте `errors.As` для проверки конкретного типа:

```go
import "errors"

result, err := api.BuyStars(ctx, "user", 100, false)
if err != nil {
    var authErr *fragment.AuthenticationError
    var userErr *fragment.UserNotFoundError
    var balErr  *fragment.InsufficientBalanceError

    switch {
    case errors.As(err, &authErr):
        log.Println("Сессия истекла, обновите cookies")
    case errors.As(err, &userErr):
        log.Printf("Пользователь не найден: %s", userErr.Username)
    case errors.As(err, &balErr):
        log.Printf("Нужно %.6f TON, есть %.6f", balErr.Required, balErr.Current)
    default:
        log.Printf("Ошибка: %v", err)
    }
}
```

### Иерархия ошибок

```
APIError (базовый)
├── AuthenticationError
├── UserNotFoundError
├── InvalidAmountError
├── InsufficientBalanceError
├── PaymentInitiationError
├── TransactionError
├── NetworkError
├── RateLimitError
└── WalletError
    └── InvalidWalletVersionError
```

---

## Статус разработки

Библиотека представляет собой **структурный скелет** (v1.0.0). HTTP-клиент,
типы, ошибки, валидация и весь высокоуровневый API реализованы и компилируются.

**Требует реализации:** методы `wallet.go` (`getBalance`, `sendTransaction`,
`transferTON`) возвращают заглушки. Интегрируйте Go TON SDK:

- [xssnick/tonutils-go](https://github.com/xssnick/tonutils-go)
- [tonkeeper/tongo](https://github.com/tonkeeper/tongo)

Каждый метод-заглушка содержит подробный TODO с псевдокодом.

---

## Лицензия

MIT. Основано на [fragment-api-py](https://github.com/S1qwy/fragment-api-py) от [S1qwy](https://github.com/S1qwy).
