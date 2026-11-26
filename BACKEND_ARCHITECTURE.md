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
│   │   ├── category/  (category.go)
│   │   ├── auth/      (entity.go, errors.go, repository.go)
│   │   ├── user/      (entity.go, errors.go, repository.go)
│   │   ├── task/      (entity.go, errors.go, repository.go)
│   │   ├── master/    (entity.go, errors.go, repository.go)
│   │   ├── bid/       (entity.go, errors.go, repository.go)
│   │   └── review/    (entity.go, errors.go, repository.go)
│   ├── usecase/
│   │   ├── auth/      (tokens.go, register.go, login.go, refresh.go, logout.go)
│   │   ├── task/      (create_task.go, list_tasks.go, update_task.go, assign_master.go, mark_completed.go, complete_task.go, cancel_task.go)
│   │   ├── master/    (upsert_profile.go, get_my_profile.go, get_public_profile.go, create_service.go, update_service.go, delete_service.go, list_services.go)
│   │   ├── bid/       (create_bid.go, list_bids.go)
│   │   └── review/    (create_review.go, update_review.go, delete_review.go)
│   ├── repository/postgres/
│   │   ├── user_repo.go
│   │   ├── refresh_token_repo.go
│   │   ├── task_repo.go
│   │   ├── master_repo.go
│   │   ├── bid_repo.go
│   │   └── review_repo.go
│   └── delivery/http/
│       ├── middleware/ (logger.go, auth.go)
│       ├── router.go
│       ├── response.go
│       ├── auth_handler.go
│       ├── task_handler.go
│       ├── master_handler.go
│       ├── bid_handler.go
│       └── review_handler.go
├── pkg/
│   ├── logger/logger.go
│   └── token/ (jwt.go, refresh.go)
├── migrations/
│   ├── 000001_create_users_table.up.sql
│   ├── 000002_create_tasks_table.up.sql
│   ├── 000003_create_master_profiles.up.sql
│   ├── 000004_create_task_bids.up.sql
│   ├── 000005_add_master_contact_phone.up.sql
│   ├── 000006_add_task_location_and_contacts.up.sql
│   ├── 000007_create_refresh_tokens_table.up.sql
│   └── 000008_rework_masters_tasks_reviews.up.sql
├── docker-compose.yml
├── go.mod
├── .env
└── .gitignore
```

---

## 1. Точки входа и конфигурация

### `cmd/api/main.go` (Composition Root)
Единственное место в программе, знающее обо всех слоях одновременно:
- Загружает конфигурацию из `.env` через `config.Load()` (при ошибке выводит сообщение в `os.Stderr` и завершает процесс с кодом `1`).
- Инициализирует логгер `logrus` и открывает пул соединений `pgxpool` к PostgreSQL.
- Создаёт экземпляры репозиториев (`UserRepo`, `RefreshTokenRepo`, `TaskRepo`, `MasterRepo`, `BidRepo`, `ReviewRepo`).
- Связывает зависимости: репозитории → юзкейсы → HTTP-хэндлеры → роутер.
- Запускает `http.Server` с таймаутами и реализует Graceful Shutdown по сигналам `SIGINT` / `SIGTERM`.

### `internal/config/config.go`
- Загружает переменные из `.env` (через `godotenv`) и системного окружения в структуру `Config` (`HTTP_PORT`, `DATABASE_URL`, `JWT_SECRET`, `ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL`, `LOG_LEVEL`, `APP_ENV`).
- Завершает запуск с ошибкой при отсутствии критически важных параметров (`JWT_SECRET`, `DATABASE_URL`).

---

## 2. Слой Домена — `internal/domain` (Core)

Самый внутренний и независимый слой. Не имеет внешних зависимостей (ни SQL, ни HTTP, ни JSON-тегов). Определяет бизнес-сущности, инварианты, доменные ошибки и интерфейсы репозиториев.

### `internal/domain/category` (Категории работ)
- **`category.go`**: общий словарь допустимых категорий (`electric`, `plumbing`, `earthworks`, `repair`, `cargo`, `other`), функции `Normalize(raw)` и `IsValid(cat)` для строгой валидации (whitelist) на уровне домена.

### `internal/domain/auth` (Refresh-токены)
- **`entity.go`**: сущность `RefreshToken` (`UserID`, `TokenHash`, `ExpiresAt`, `RevokedAt`) с методами `IsRevoked()` и `IsExpired(now)`.
- **`errors.go`**: `ErrInvalidRefreshToken`, `ErrRefreshTokenExpired`, `ErrRefreshTokenRevoked`.
- **`repository.go`**: интерфейс `RefreshTokenRepository` (`Create`, `GetByHash`, `RevokeIfActive`, `RevokeByHash`, `RevokeAllByUserID`).

### `internal/domain/user` (Пользователи)
- **`entity.go`**: структура `User` и конструктор `NewUser`. Нормализует email (`TrimSpace + ToLower`), очищает пробелы в имени и телефоне, запрещает пустое имя или некорректный email.
- **`errors.go`**: `ErrUserNotFound`, `ErrEmailAlreadyExists`, `ErrInvalidCredentials`, `ErrInvalidEmail`.
- **`repository.go`**: интерфейс `Repository` с методами `Create`, `GetByID`, `GetByEmail`, `Update`.

### `internal/domain/task` (Объявления)
- **`entity.go`**: богатая доменная модель `Task` со State Machine статусов (`open → in_progress → pending_completion → completed / canceled`), ссылкой `AssignedMasterID` на `master_profiles.id` и полями `District`, `ContactPhone`, `WorkAddress`:
  - `NewTask` и `Update`: переиспользуют хелпер `validateAndNormalizeFields`.
  - `AssignMaster(actorID, masterProfileID, masterUserID)`: проверяет права автора, запрещает самоназначение (`t.AuthorID != masterUserID`) и переводит статус в `in_progress`.
  - `MarkCompletedByMaster(actorProfileID)`: переводит задачу из `in_progress` в `pending_completion` назначенным мастером.
  - `Complete(actorID)`: закрывает задачу из `in_progress` или `pending_completion` автором (при отсутствии исполнителя возвращает `ErrMasterNotAssigned`).
  - `Cancel(actorID)`: отменяет незавершённую задачу, блокируя повторную отмену.
- **`errors.go`**: `ErrTaskNotFound`, `ErrForbiddenAction`, `ErrInvalidStatus`, `ErrSelfAssignment`, `ErrInvalidMaster`, `ErrMasterNotAssigned`, `ErrDistrictTooLong`, `ErrInvalidPhone`, `ErrAddressTooLong`, `ErrEmptyCategory`, `ErrInvalidCategory`.
- **`repository.go`**: интерфейс `Repository` (`Create`, `GetByID`, `List`, `Update(ctx, t, allowedPrevStatuses...)`, `CompleteAndIncrementMasterCounter`).

### `internal/domain/master` (Профили мастеров и анкеты услуг)
- **`entity.go`**:
  - `Profile` (1 на `user_id`): `DisplayName`, `AvatarURL` (с проверкой `http://`/`https://`), `About`, `City`, `District`, `ContactPhone` (`+7XXXXXXXXXX`), `Telegram`, `IsContactsPublic`, агрегаты `RatingAvg`, `ReviewsCount`, `CompletedOrdersCount`.
  - `Service` (до `MaxActiveServices = 3` активных на профиль): `Category` (whitelist), `Title`, `ExperienceDescription`, `Price`, `IsActive`.
  - `ServiceCatalogItem`: read-модель каталога услуг с метриками профиля мастера.
- **`errors.go`**: `ErrProfileNotFound`, `ErrProfileAlreadyExists`, `ErrServiceNotFound`, `ErrServiceCategoryExists`, `ErrMaxActiveServicesReached`, `ErrEmptyDisplayName`, `ErrDisplayNameTooLong`, `ErrCityTooLong`, `ErrTelegramTooLong`, `ErrInvalidAvatarURL`, `ErrEmptyTitle`, `ErrTitleTooShort`, `ErrTitleTooLong`, `ErrInvalidPrice`, `ErrEmptyContactPhone`, `ErrInvalidPhone`, `ErrDistrictTooLong`, `ErrForbiddenAction`, `ErrProfileInUse`, `ErrEmptyCategory`, `ErrInvalidCategory`.
- **`repository.go`**: методы профиля (`CreateProfile`, `GetProfileByID`, `GetProfileByUserID`, `UpdateProfile`, `DeleteProfile`, `HasSharedTaskWithProfile`) и услуг (`CreateService`, `GetServiceByID`, `UpdateService`, `DeleteService`, `CountActiveServices`, `ListServicesByProfileID`, `ListCatalogServices`).

### `internal/domain/bid` (Отклики мастеров)
- **`entity.go`**: сущность `Bid` (`task_responses`) с `MasterProfileID`, опциональными `ProposedPrice (*int64)` и `Comment`, а также краткими метриками профиля мастера для списка откликов.
- **`errors.go`**: `ErrBidAlreadyExists`, `ErrSelfBidNotAllowed`, `ErrTaskNotOpenForBids`, `ErrMasterProfileNeeded`, `ErrForbiddenView`.
- **`repository.go`**: интерфейс `Repository` (`Create`, `ListByTaskID`).

### `internal/domain/review` (Отзывы и рейтинг)
- **`entity.go`**: сущность `Review` (`TaskID`, `MasterProfileID`, `AuthorUserID`, `Rating 1..5`, `Comment`) с конструктором `NewReview` и методом `Update`.
- **`errors.go`**: `ErrInvalidTaskID`, `ErrInvalidMasterID`, `ErrInvalidAuthorID`, `ErrReviewNotFound`, `ErrReviewAlreadyExists`, `ErrInvalidRating`, `ErrTaskNotCompleted`, `ErrForbiddenReview`.
- **`repository.go`**: интерфейс `Repository` (`Create`, `GetByID`, `GetByTaskID`, `GetByTaskIDs`, `ListByMasterProfileID`, `Update`, `Delete`).

---

## 3. Слой Бизнес-сценариев — `internal/usecase` (Application)

### `internal/usecase/auth`
- **`tokens.go`**, **`register.go`**, **`login.go`**, **`refresh.go`**, **`logout.go`**: выпуск пары JWT access + refresh (SHA-256), атомарная ротация с отзывом всех сессий при повторном использовании, выход.

### `internal/usecase/task`
- **`create_task.go`**, **`list_tasks.go`**, **`update_task.go`**, **`cancel_task.go`**: базовый жизненный цикл объявлений.
- **`assign_master.go` (`AssignMasterUseCase`)**: проверяет существование `master_profiles` по `MasterProfileID` и запрещает самоназначение.
- **`mark_completed.go` (`MarkTaskCompletedUseCase`)**: перевод назначенным мастером `in_progress → pending_completion`.
- **`complete_task.go` (`CompleteTaskUseCase`)**: закрытие автором (`in_progress` / `pending_completion → completed`) через `CompleteAndIncrementMasterCounter`.

### `internal/usecase/master`
- **`upsert_profile.go` (`UpsertProfileUseCase`)**: создание или обновление профиля мастера (с поддержкой `*bool` для `IsContactsPublic` и фоллбэком пустого `DisplayName` на `user.Name`).
- **`get_my_profile.go` (`GetMyProfileUseCase`)**: возвращает свой профиль и все услуги (включая скрытые).
- **`get_public_profile.go` (`GetPublicProfileUseCase`)**: возвращает публичный профиль, активные услуги, отзывы и фильтрует контакты на бэкенде при `is_contacts_public == false`.
- **`create_service.go`**, **`update_service.go`**, **`delete_service.go`**, **`list_services.go`**: CRUD анкет услуг и публичный каталог.

### `internal/usecase/bid`
- **`create_bid.go` (`CreateBidUseCase`)**: проверяет наличие профиля и `CountActiveServices > 0`, статус задачи (`open`), запрет самоотклика.
- **`list_bids.go` (`ListBidsUseCase`)**: возвращает отклики автору задачи.

### `internal/usecase/review`
- **`create_review.go`**, **`update_review.go`**, **`delete_review.go`**: создание (только автор, только `completed` заказ), редактирование и удаление отзыва.

---

## 4. Слой Хранения — `internal/repository/postgres` (Infrastructure)

- **`user_repo.go` (`UserRepo`)**: работа с `users`.
- **`refresh_token_repo.go` (`RefreshTokenRepo`)**: работа с `refresh_tokens`.
- **`task_repo.go` (`TaskRepo`)**: `assigned_master_id`, оптимистическая проверка `allowedPrevStatuses` в `Update` и транзакционное завершение `CompleteAndIncrementMasterCounter` с `FOR UPDATE`.
- **`master_repo.go` (`MasterRepo`)**: CRUD профилей и услуг, блокировка профиля `FOR UPDATE` при создании/активации услуги для соблюдения лимита `3`, маппинг `pgErr.ConstraintName` (`23505` / `23503` / `23514`), проверка совместных заказов `HasSharedTaskWithProfile`.
- **`bid_repo.go` (`BidRepo`)**: работа с `task_responses` с `JOIN master_profiles`.
- **`review_repo.go` (`ReviewRepo`)**: CRUD по `master_reviews` с блокировкой профиля `FOR UPDATE` и транзакционным пересчётом `rating_avg` / `reviews_count`, батч-загрузка `GetByTaskIDs` (`ANY($1)`) и пагинация `ListByMasterProfileID`.

---

## 5. Транспортный слой — `internal/delivery/http` (Delivery)

- **`middleware/logger.go`**, **`middleware/auth.go`** (`Auth`, `OptionalAuth`), **`response.go`**.
- **`router.go` (`NewRouter`)**: публичные маршруты `/health`, `/api/v1/auth/*`, `GET /tasks`, `GET /masters`, `GET /masters/{id}`, `GET /master-services` (с `optAuthMW`), приватные маршруты с `authMW` и статика `web/` с `Cache-Control: no-cache`.
- **`auth_handler.go`**: `POST /auth/register`, `/login`, `/refresh`, `/logout`.
- **`task_handler.go`**: `POST/GET /tasks`, `PUT /tasks/{id}`, `POST /tasks/{id}/assign`, `/mark-completed`, `/complete`, `/cancel` (валидация `limit`/`offset`/`status`, батч-загрузка `my_review` без N+1).
- **`master_handler.go`**: `GET/PUT /api/v1/master-profile/me`, `GET /api/v1/masters/{id}`, `GET /api/v1/master-services` (и alias `/masters`), `POST/PUT/DELETE /api/v1/master-services/{id}`.
- **`bid_handler.go`**: `POST/GET /api/v1/tasks/{id}/bids`.
- **`review_handler.go`**: `POST /api/v1/tasks/{id}/review`, `PUT/DELETE /api/v1/reviews/{id}`.

---

## 6. Утилиты и инфраструктура (`pkg`, `migrations`, Docker)

- **`pkg/logger/logger.go`**, **`pkg/token/jwt.go`**, **`pkg/token/refresh.go`**.
- **`migrations/`**:
  1–7. Базовые таблицы (`users`, `tasks`, `master_profiles`, `task_bids`, контакты, `refresh_tokens`).
  8. `000008_rework_masters_tasks_reviews.up.sql`: разделение на `master_profiles` и `master_services` (с переносом данных), замена `tasks.master_id` на `assigned_master_id`, перенос `task_bids → task_responses`, создание `master_reviews`.
- **`docker-compose.yml`**, **`go.mod`**, **`.env`**, **`.gitignore`**.
