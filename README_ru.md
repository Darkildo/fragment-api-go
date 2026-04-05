# Fragment API Go

[![Go 1.26+](https://img.shields.io/badge/go-1.26+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/Darkildo/fragment-api-go)

**Go-клиент для Fragment.com API — Telegram Stars, Premium-подписки и TON-переводы.**

Go-порт [fragment-api-py](https://github.com/S1qwy/fragment-api-py) (Python v3.2.0).  
Интеграция с TON через [tonutils-go](https://github.com/xssnick/tonutils-go).

[README in English](README.md)

---

## Установка

```bash
go get github.com/Darkildo/fragment-api-go
```

## Быстрый старт

```go
import fragment "github.com/Darkildo/fragment-api-go"

api, err := fragment.New(fragment.Config{
    Cookies:        "stel_ssid=...; stel_token=...; stel_dt=...; stel_ton_token=...",
    HashValue:      "ваш_hash_из_network_tab",
    WalletMnemonic: "word1 word2 ... word24",
})
if err != nil {
    log.Fatal(err)
}
defer api.Close()

ctx := context.Background()

// Отправить 100 Stars
result, err := api.BuyStars(ctx, "jane_doe", 100, false)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("TX: %s\n", result.TransactionHash)
```

## Возможности

- **Telegram Stars** — отправка Stars любому пользователю
- **Premium-подарки** — подписка Telegram Premium на 3/6/12 месяцев
- **Пополнение TON Ads** — пополнение рекламных аккаунтов
- **Прямые TON-переводы** — отправка TON на любой адрес с комментарием
- **Мульти-кошелек** — V3R1, V3R2, V4R2 (по умолчанию), V5R1/W5 как типизированный enum
- **Видимость отправителя** — анонимные или видимые платежи
- **Автоматические повторы** — экспоненциальный backoff с поддержкой context
- **Сериализация транзакций** — безопасный вызов из нескольких горутин; транзакции ставятся в очередь и выполняются по одной через канал-семафор с поддержкой отмены через context
- **Типизированные ошибки** — полная цепочка через `errors.Is` / `errors.As`; отдельный `TransactionNotConfirmedError` для таймаутов
- **Структурное логирование** — опциональное через `log/slog` (stdlib, без зависимостей)
- **Минимум зависимостей** — только [tonutils-go](https://github.com/xssnick/tonutils-go) + stdlib

## Структура проекта

```
fragment-api-go/
  go.mod           определение модуля
  fragment.go      Client, New(), Config, Close, WalletBalance, WalletInfo
  types.go         UserInfo, PurchaseResult, TransferResult, WalletBalance, WalletVersion enum, WalletInfo
  errors.go        иерархия APIError (10 типов с цепочками Unwrap)
  recipient.go     GetRecipientStars, GetRecipientPremium, GetRecipientTON
  stars.go         BuyStars
  premium.go       GiftPremium
  topup.go         TopupTON
  transfer.go      TransferTON
  purchase.go      общий флоу покупки (неэкспортируемый)
  core.go          HTTP-транспорт (неэкспортируемый)
  wallet.go        TON-кошелек через tonutils-go (неэкспортируемый)
  helpers.go       валидация, парсинг, конвертация (неэкспортируемый)
  LICENSE          MIT
  example/main.go  пример использования
```

---

## API-справочник

```go
api, err := fragment.New(fragment.Config{ ... })
defer api.Close()

// Поиск получателя
user, err := api.GetRecipientStars(ctx, "username")
user, err := api.GetRecipientPremium(ctx, "username")
user, err := api.GetRecipientTON(ctx, "username")

// Покупки — возвращают (*PurchaseResult, error)
result, err := api.BuyStars(ctx, "username", 100, false)
result, err := api.GiftPremium(ctx, "username", 3, false)
result, err := api.TopupTON(ctx, "username", 10, false)

// Прямой перевод — возвращает (*TransferResult, error)
transfer, err := api.TransferTON(ctx, "EQ...", 0.5, "memo")

// Кошелек
balance, err := api.WalletBalance(ctx)   // *WalletBalance
info := api.WalletInfo()                 // WalletInfo (типизированная структура)
```

---

## Версии кошельков (типизированный Enum)

```go
fragment.WalletV3R1  // "V3R1" — legacy
fragment.WalletV3R2  // "V3R2" — legacy
fragment.WalletV4R2  // "V4R2" — по умолчанию, рекомендуется
fragment.WalletV5R1  // "V5R1" — новейший
fragment.WalletW5    // "W5"   — алиас для V5R1
```

Config принимает регистронезависимые строки: `"v4r2"`, `"V4R2"`, `"w5"`.

---

## Обработка ошибок

Все ошибки формируют цепочку. Используйте `errors.As` / `errors.Is` для типизированного сопоставления.
Ошибки никогда не теряются в строковых полях — всегда возвращаются как Go-ошибки.

```go
result, err := api.BuyStars(ctx, "user", 100, false)
if err != nil {
    var notConfirmed *fragment.TransactionNotConfirmedError
    var txErr  *fragment.TransactionError
    var balErr *fragment.InsufficientBalanceError
    var userErr *fragment.UserNotFoundError

    switch {
    case errors.As(err, &notConfirmed):
        // TX отправлена, но не подтверждена — может подтвердиться позже!
        // Проверьте состояние блокчейна перед повтором (double-spend).
        log.Printf("TX в ожидании: %v", notConfirmed)
    case errors.As(err, &txErr):
        log.Printf("TX ошибка: %v", txErr)
    case errors.As(err, &balErr):
        log.Printf("Нужно %.6f TON, есть %.6f", balErr.Required, balErr.Current)
    case errors.As(err, &userErr):
        log.Printf("Пользователь %q не найден", userErr.Username)
    default:
        log.Printf("Ошибка: %v", err)
    }
}
```

### Иерархия ошибок

```
APIError (базовый, имеет Unwrap)
├── AuthenticationError
├── UserNotFoundError              — .Username
├── InvalidAmountError             — .Amount, .MinValue, .MaxValue
├── InsufficientBalanceError       — .Required, .Current
├── PaymentInitiationError
├── TransactionError
│   └── TransactionNotConfirmedError — tx отправлена, но не подтверждена за deadline
├── NetworkError                   — .StatusCode
├── RateLimitError                 — .RetryAfter
└── WalletError
    └── InvalidWalletVersionError  — .Version, .SupportedVersions
```

---

## Логирование

Передайте `*slog.Logger` для структурного логирования (stdlib `log/slog`).
`nil` отключает логирование полностью (nop-handler, нулевой overhead).

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

api, _ := fragment.New(fragment.Config{
    // ...
    Logger: logger,
})
```

---

## Конкурентность

Все методы `Client` безопасны для вызова из нескольких горутин.
Транзакции (Stars, Premium, TON-переводы) автоматически сериализуются
через внутренний семафор — только одна транзакция в блокчейне за раз.
Ожидающие горутины могут отменить ожидание через context.

```go
api, _ := fragment.New(fragment.Config{...})
defer api.Close()

var wg sync.WaitGroup
for _, user := range []string{"alice_t", "bob_smith", "charlie_99"} {
    wg.Add(1)
    go func(username string) {
        defer wg.Done()
        ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
        defer cancel()

        // Безопасно: параллельные вызовы ставятся в очередь.
        // Каждая транзакция ждёт завершения предыдущей.
        result, err := api.BuyStars(ctx, username, 100, false)
        if err != nil {
            log.Printf("%s: %v", username, err)
            return
        }
        log.Printf("%s: TX %s", username, result.TransactionHash)
    }(user)
}
wg.Wait()
```

**Почему сериализация?** TON-кошельки используют sequence number (seqno),
который увеличивается с каждой исходящей транзакцией. Параллельные отправки
с одного кошелька используют один seqno — одна транзакция будет отклонена сетью.
Семафор гарантирует последовательное выполнение на уровне кошелька.

**Поведение при таймауте:** если транзакция зависла (проблемы сети),
семафор удерживается до deadline context или 180-секундного fallback
из tonutils-go. Другие горутины получают `context.DeadlineExceeded`
из своего context. Зависшая транзакция никогда не блокирует семафор навсегда.

---

## Параметры Config

| Поле | Тип | По умолчанию | Описание |
|------|-----|-------------|----------|
| `Cookies` | `string` | обязательный | Session cookies Fragment.com |
| `HashValue` | `string` | обязательный | API hash из DevTools |
| `WalletMnemonic` | `string` | обязательный | 24-словная TON мнемоника |
| `WalletVersion` | `string` | `"V4R2"` | Версия кошелька (регистронезависимо) |
| `Testnet` | `bool` | `false` | Использовать TON testnet |
| `Timeout` | `time.Duration` | `15s` | HTTP таймаут для Fragment API |
| `Logger` | `*slog.Logger` | `nil` (отключено) | Структурный логгер |

---

## Лицензия

MIT. См. [LICENSE](LICENSE).

Основано на [fragment-api-py](https://github.com/S1qwy/fragment-api-py) от [S1qwy](https://github.com/S1qwy).
