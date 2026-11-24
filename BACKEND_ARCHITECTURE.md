# Bboard Backend — Архитектура (Clean Architecture & DDD)

- **Стек:** Go 1.22+ / PostgreSQL 16 (`jackc/pgx/v5`)
- **Архитектура:** Clean Architecture + Domain-Driven Design (DDD)
- **Правило зависимостей:** `Outer → Inner` (внутренние слои `domain` и `usecase` не импортируют внешние пакеты и работают только через интерфейсы).

---

## Дерево проекта

```text
bboard/
├── cmd/api/main.go
├── internal/
│   ├── config/config.go
│   ├── domain/
│   │   ├── user/      (entity.go, errors.go, repository.go)
│   │   ├── task/      (entity.go, errors.go, repository.go)
│   │   ├── master/    (entity.go, errors.go, repository.go)
│   │   └── bid/       (entity.go, errors.go, repository.go)
│   ├── usecase/
│   │   ├── auth/      (register.go, login.go)
│   │   ├── task/      (create_task.go, list_tasks.go, assign_master.go, complete_task.go, cancel_task.go)
│   │   ├── master/    (create_profile.go, list_profiles.go)
│   │   └── bid/       (create_bid.go, list_bids.go)
│   ├── repository/postgres/
│   │   ├── user_repo.go
│   │   ├── task_repo.go
│   │   ├── master_repo.go
│   │   └── bid_repo.go
│   └── delivery/http/
│       ├── middleware/ (logger.go, auth.go)
│       ├── router.go
│       ├── response.go
│       ├── auth_handler.go
│       ├── task_handler.go
│       ├── master_handler.go
│       └── bid_handler.go
├── pkg/
│   ├── logger/logger.go
│   └── token/jwt.go
├── migrations/
│   ├── 000001_create_users_table.up.sql
│   ├── 000002_create_tasks_table.up.sql
│   ├── 000003_create_master_profiles.up.sql
│   ├── 000004_create_task_bids.up.sql
│   ├── 000005_add_master_contact_phone.up.sql
│   └── 000006_add_task_location_and_contacts.up.sql
├── docker-compose.yml
├── go.mod
├── .env
└── .gitignore
```

---

## 1. Точки входа и конфигурация

### `cmd/api/main.go` (Composition Root)
Единственное место в программе, знающее обо всех слоях одновременно:
- Загружает конфигурацию из `.env` через `config.Load()`.
- Инициализирует логгер `logrus` и открывает пул соединений `pgxpool` к PostgreSQL.
- Создаёт экземпляры репозиториев (`UserRepo`, `TaskRepo`, `MasterRepo`, `BidRepo`).
- Связывает зависимости: репозитории → юзкейсы → HTTP-хэндлеры → роутер.
- Запускает `http.Server` с таймаутами и реализует Graceful Shutdown по сигналам `SIGINT` / `SIGTERM`.
- **Зависимости:** `internal/config/config.go`, `internal/repository/postgres`, `internal/usecase`, `internal/delivery/http/router.go`, `pkg/logger/logger.go`, `pkg/token/jwt.go`.

### `internal/config/config.go`
- Загружает переменные из `.env` (через `godotenv`) и системного окружения в структуру `Config` (`HTTP_PORT`, `DATABASE_URL`, `JWT_SECRET`, `JWT_TTL`, `LOG_LEVEL`, `APP_ENV`).
- Завершает запуск с ошибкой при отсутствии критически важных параметров (`JWT_SECRET`, `DATABASE_URL`).

---

## 2. Слой Домена — `internal/domain` (Core)

Самый внутренний и независимый слой. Не имеет внешних зависимостей (ни SQL, ни HTTP, ни JSON-тегов). Определяет бизнес-сущности, инварианты, доменные ошибки и интерфейсы репозиториев.

### `internal/domain/user` (Пользователи)
- **`entity.go`**: структура `User` и конструктор `NewUser`. Нормализует email (`TrimSpace + ToLower`), очищает пробелы в имени и телефоне, запрещает пустое имя или некорректный email.
- **`errors.go`**: `ErrUserNotFound`, `ErrEmailAlreadyExists`, `ErrInvalidCredentials`, `ErrInvalidEmail`.
- **`repository.go`**: интерфейс `Repository` с методами `Create`, `GetByID`, `GetByEmail`.

### `internal/domain/task` (Объявления)
- **`entity.go`**: богатая доменная модель `Task` со State Machine статусов (`open → in_progress → completed / canceled`) и полями `District`, `ContactPhone`, `WorkAddress`:
  - `NewTask`: валидирует заголовок, описание, категорию, неотрицательный бюджет, длину района (до 100 симв.), нормализует телефон (замена 8 на +7, очистка скобок/дефисов/пробелов, проверка `^\+7\d{10}$`) и длину адреса (до 255 симв.).
  - `AssignMaster(actorID, masterID)`: проверяет права автора, запрещает самоназначение и переводит статус в `in_progress`.
  - `Complete(actorID)`: закрывает задачу только из статуса `in_progress` при наличии назначенного мастера.
  - `Cancel(actorID)`: отменяет открытую или выполняемую задачу, блокируя повторную отмену.
- **`errors.go`**: `ErrTaskNotFound`, `ErrForbiddenAction`, `ErrInvalidStatus`, `ErrSelfAssignment`, `ErrInvalidMaster`, `ErrDistrictTooLong`, `ErrInvalidPhone`, `ErrAddressTooLong`.
- **`repository.go`**: интерфейс `Repository` (`Create`, `GetByID`, `List(ctx, filter)`, `Update`) и структура `ListFilter`.

### `internal/domain/master` (Анкеты мастеров)
- **`entity.go`**: структура `Profile`, конструктор `NewProfile` и метод `Update`. Проверяет минимальную длину заголовка, неотрицательный стаж (`ExperienceYears >= 0`) и минимальную цену (`MinPrice >= 0`), управляет флагом `IsActive`.
- **`errors.go`**: `ErrProfileNotFound`, `ErrProfileAlreadyExists`, `ErrInvalidExperience`, `ErrInvalidMinPrice`.
- **`repository.go`**: интерфейс `Repository` (`Create`, `GetByID`, `GetByUserID`, `List`, `Update`).

### `internal/domain/bid` (Отклики мастеров)
- **`entity.go`**: сущность `Bid` с предложенной ценой (`ProposedPrice`) и комментарием. Валидирует ID задачи, ID мастера, неотрицательную цену и длину комментария (минимум 5 символов).
- **`errors.go`**: `ErrBidAlreadyExists`, `ErrSelfBidNotAllowed`, `ErrTaskNotOpenForBids`, `ErrMasterProfileNeeded`, `ErrForbiddenView`.
- **`repository.go`**: интерфейс `Repository` (`Create`, `ListByTaskID`).

---

## 3. Слой Бизнес-сценариев — `internal/usecase` (Application)

Оркестрирует работу доменных репозиториев и инфраструктурных сервисов (`bcrypt`, `JWT`).

### `internal/usecase/auth`
- **`register.go` (`RegisterUseCase`)**: проверяет длину пароля (`>= 6` символов) и уникальность email, хеширует пароль через `bcrypt.DefaultCost`, создаёт `user.NewUser` и сохраняет в `userRepo`.
- **`login.go` (`LoginUseCase`)**: находит пользователя по email, сверяет bcrypt-хеш и выпускает JWT-токен через `TokenGenerator`.

### `internal/usecase/task`
- **`create_task.go` (`CreateTaskUseCase`)**: проверяет существование автора в `userRepo`, создаёт задачу в статусе `open` и сохраняет в `taskRepo`.
- **`list_tasks.go` (`ListTasksUseCase`)**: нормализует `limit`/`offset`, по умолчанию фильтрует задачи по `status=open`.
- **`assign_master.go` (`AssignMasterUseCase`)**: проверяет через `masterRepo` наличие активной анкеты у исполнителя, вызывает `t.AssignMaster(actorID, masterID)` и сохраняет задачу.
- **`complete_task.go` (`CompleteTaskUseCase`)**: загружает задачу, вызывает `t.Complete(actorID)` и обновляет запись в БД.
- **`cancel_task.go` (`CancelTaskUseCase`)**: загружает задачу, вызывает `t.Cancel(actorID)` и сохраняет статус `canceled`.

### `internal/usecase/master`
- **`create_profile.go` (`CreateProfileUseCase`)**: проверяет существование пользователя, валидирует профиль через `master.NewProfile` и сохраняет его.
- **`list_profiles.go` (`ListProfilesUseCase`)**: фильтрует анкеты по категории и городу, возвращая только профили с `IsActive=true`.

### `internal/usecase/bid`
- **`create_bid.go` (`CreateBidUseCase`)**: проверяет наличие активной анкеты мастера у пользователя, статус задачи (`open`) и отсутствие самоотклика; сохраняет отклик в `bidRepo`.
- **`list_bids.go` (`ListBidsUseCase`)**: проверяет, что запрашивающий является автором задачи (`t.AuthorID == input.ActorID`), иначе возвращает `ErrForbiddenView`.

---

## 4. Слой Хранения — `internal/repository/postgres` (Infrastructure)

Реализует интерфейсы из `internal/domain/` на базе `pgxpool` (`pgx/v5`), маппит строки БД в доменные сущности и преобразует SQL-ошибки (например, `SQLSTATE 23505` — нарушение уникальности) в доменные ошибки.

- **`user_repo.go` (`UserRepo`)**: `INSERT` и `SELECT` по `id` и `email` в таблице `users`.
- **`task_repo.go` (`TaskRepo`)**: работа с таблицей `tasks`, поддержка nullable-колонки `master_id (*int64)` и динамическое построение `WHERE` для фильтров.
- **`master_repo.go` (`MasterRepo`)**: работа с таблицей `master_profiles`, поиск по `user_id` и выборка каталога с сортировкой по опыту.
- **`bid_repo.go` (`BidRepo`)**: запись откликов в `task_bids` (с проверкой уникальности пары `task_id + master_id`) и выборка откликов по задаче.

---

## 5. Транспортный слой — `internal/delivery/http` (Delivery)

Реализация REST API на стандартной библиотеке Go 1.22+ (`http.ServeMux` с поддержкой методов и path-параметров `{id}`).

- **`middleware/logger.go`**: генерирует уникальный `X-Request-ID`, перехватывает HTTP-статус и замеряет `duration_ms` для `logrus`.
- **`middleware/auth.go`**: содержит `Auth` (строгая проверка Bearer-токена с 401) и `OptionalAuth` (извлекает `userID` из валидного Bearer-токена, пропуская гостей без токена без ошибки 401).
- **`router.go` (`NewRouter`)**: регистрирует публичные маршруты (`/health`, `/api/v1/auth/*`, `GET /tasks`, `GET /masters`) и оборачивает приватные маршруты в `authMW`.
- **`response.go`**: хелпер `writeJSON` с заголовком `Content-Type: application/json` и единым форматом `errorResponse`.
- **`auth_handler.go`**: регистрация и вход (использует `MaxBytesReader` и `DisallowUnknownFields`).
- **`task_handler.go`**: `POST /tasks`, `GET /tasks`, `POST /tasks/{id}/assign`, `/complete`, `/cancel`. Район (`district`) отдаётся всем, а `contact_phone` и `work_address` — только автору и назначенному мастеру в статусах `in_progress`/`completed`. Маппит доменные ошибки в HTTP-коды (`400`, `403`, `404`, `409`).
- **`master_handler.go`**: `POST /api/v1/masters` и `GET /api/v1/masters`.
- **`bid_handler.go`**: `POST /api/v1/tasks/{id}/bids` и `GET /api/v1/tasks/{id}/bids`.

---

## 6. Утилиты и инфраструктура (`pkg`, `migrations`, Docker)

- **`pkg/logger/logger.go`**: конфигурация `*logrus.Logger` (цветной текстовый вывод в `dev`, структурированный JSON в `prod`).
- **`pkg/token/jwt.go` (`JWTManager`)**: выпуск (`Generate(userID)`) и проверка (`Parse(tokenStr)`) токенов по алгоритму `HS256`.
- **`migrations/`** (для `golang-migrate`, пары `.up.sql` / `.down.sql`):
  1. `000001_create_users_table.up.sql`: таблица `users` (`id`, `name`, `email UNIQUE`, `phone`, `password_hash`, `created_at`, `updated_at`).
  2. `000002_create_tasks_table.up.sql`: таблица `tasks`, внешние ключи `author_id` и `master_id`, составной индекс `(status, category, created_at DESC)`.
  3. `000003_create_master_profiles.up.sql`: таблица `master_profiles` с ограничением `UNIQUE (user_id)`.
  4. `000004_create_task_bids.up.sql`: таблица `task_bids` с ограничением `UNIQUE (task_id, master_id)`.
  5. `000005_add_master_contact_phone.up.sql`: таблица `master_profiles` (колонка `contact_phone`).
  6. `000006_add_task_location_and_contacts.up.sql`: таблица `tasks` (колонки `district`, `contact_phone`, `work_address` с default '').
- **`docker-compose.yml`**: поднимает `postgres` (PostgreSQL 16-alpine с healthcheck и volume) и одноразовый контейнер `migrate`.
- **`go.mod`**: прямые зависимости (`jackc/pgx/v5`, `golang-jwt/jwt/v5`, `sirupsen/logrus`, `joho/godotenv`, `golang.org/x/crypto`).
- **`.env` / `.gitignore`**: локальные секреты окружения и защита репозитория от утечки `.env` и бинарников.
