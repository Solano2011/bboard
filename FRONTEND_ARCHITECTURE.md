# Bboard Frontend — Архитектура (Flat Design 2.0 & ES6+)

- **Стек:** Vanilla ES6+ Modules, HTML5, CSS3 Variables (Zero-Build SPA)
- **Раздача:** напрямую Go-сервером через `http.FileServer(http.Dir("web"))` на том же домене и порту (`:8080`), что исключает проблемы с CORS.
- **Архитектура:** Modular SPA + Unidirectional Data Flow (`Action → API → Store → Reactive UI`).

---

## Дерево фронтенда (`web/`)

```text
web/
├── index.html
├── css/
│   └── styles.css
└── js/
    ├── api/
    │   └── httpClient.js
    ├── constants/
    │   └── categories.js
    ├── store/
    │   └── store.js
    ├── components/
    │   ├── taskCard.js
    │   ├── masterCard.js
    │   └── modals.js
    └── app.js
```

---

## 1. Каркас и стилизация

### `web/index.html`
- **Паттерн:** Application Shell (Каркас приложения).
- Единственная HTML-страница SPA, изолирующая разметку от бизнес-логики.
- Подключает `/css/styles.css` и точку входа `/js/app.js` (`type="module"`).
- Содержит вкладки навигации (`Лента заказов`, `Каталог мастеров`, `Мой кабинет` для авторизованных), фильтр статусов (включая `pending_completion`) и семантические контейнеры (`#navActions`, `#primaryCta`, `#contentGrid`, `#modalBackdrop`).

### `web/css/styles.css`
- **Паттерн:** Design Tokens (CSS Custom Properties) + BEM-like Component Classes.
- Реализует дизайн-систему **Flat Design 2.0** через CSS-переменные в `:root`.
- Определяет пастельные бейджи статусов (`.status-open`, `.status-in_progress`, `.status-pending_completion`, `.status-completed`, `.status-canceled`), сетку карточек (`.grid`), модальные окна (`.modal-backdrop`, `.modal-dialog`), уведомления (`.toast`), а также стили личного кабинета мастера (`.cabinet-grid`, `.cabinet-panel`, `.service-item`, `.switch-row`) и публичного профиля/отзывов (`.profile-header`, `.profile-avatar`, `.rating-badge`, `.review-item`, `.task-review-box`).

---

## 2. Слой клиентской логики (`web/js/`)

### `web/js/api/httpClient.js` (Слой транспорта)
- **Паттерн:** Facade + Token Provider Injection (Dependency Inversion).
- Единый адаптер поверх `Fetch API` для запросов к `/api/v1/*` с `credentials: "include"` для работы с `HttpOnly` cookie `refresh_token`.
- Перехватывает `401 Unauthorized` (кроме `/auth/*`), выполняет одиночный запрос `POST /api/v1/auth/refresh` через общий промис `refreshPromise` (защита от race condition), обновляет `access_token` через `tokenUpdatedHandler` и повторяет исходный запрос; при отказе refresh вызывает `unauthorizedHandler`.
- Методы:
  - Авторизация: `login`, `register`, `logout`, `me`.
  - Заказы и отклики: `listTasks`, `createTask`, `updateTask`, `assignMaster(taskId, masterProfileId)`, `markTaskCompleted`, `completeTask`, `cancelTask`, `createBid`, `listTaskBids`.
  - Мастера и услуги: `getMyMasterProfile`, `upsertMyMasterProfile`, `getPublicMasterProfile`, `listMasterServices`, `createMasterService`, `updateMasterService`, `deleteMasterService`.
  - Отзывы: `createReview`, `updateReview`, `deleteReview`.

### `web/js/constants/categories.js` (Справочник категорий)
- Единый источник истины для списка категорий (`CATEGORIES`), словаря названий (`CATEGORY_LABELS`) и генератора `<option>` (`renderCategoryOptions`).

### `web/js/store/store.js` (Слой состояния)
- **Паттерн:** Observer (Pub-Sub) + Singleton + Web Storage Persistence.
- Хранит сессию (`token`, `user`), активную вкладку (`tasks` | `masters` | `cabinet`), фильтры, `tasks`, `masters`, `myProfile` и `myServices`.
- `clearSession()` очищает `localStorage`, сбрасывает `myProfile`/`myServices` и переводит с вкладки `cabinet` на `tasks`.

### `web/js/components/` (Слой представления)
- **`taskCard.js` (`renderTaskCard`)**:
  - Поддерживает статус `pending_completion` («Ожидает подтверждения»).
  - Отображает кнопки действий по роли: «Отклики и выбор мастера» / «Редактировать» / «Отменить» для автора (`open`), «Откликнуться» для других пользователей (`open`), «Отметить выполненным» для назначенного мастера (`in_progress`), «Завершить работу» / «Подтвердить выполнение» для автора, а также блок отзыва (`my_review`) и кнопки «Оставить отзыв» / «Редактировать отзыв» / «Удалить отзыв» после завершения заказа.
  - При наличии `assigned_master_id` добавляет кнопку «Профиль исполнителя».
- **`masterCard.js` (`renderMasterCard`, `formatRating`)**:
  - Рендерит карточку услуги каталога (`ServiceCatalogItem`): категория, рейтинг (`★ 5.0 (3)` или `Нет отзывов`), заголовок, опыт, имя мастера, счётчик выполненных заказов, город/район, цену и кнопку «Профиль мастера».
- **`modals.js` (`ModalManager` & `showToast`)**:
  - Управляет `#modalBackdrop` и всплывающими уведомлениями.

### `web/js/app.js` (Контроллер приложения)
- **Паттерн:** Mediator + Event Delegation.
- Последовательная инициализация `initApp()`: дожидается `validateSessionOnStartup()` (обновление протухшего access-токена через `api.me()`) до первичной загрузки `loadData()`.
- Реализует вкладку **«Мой кабинет»** (`renderCabinet`): форма профиля мастера (включая тумблер `is_contacts_public`) и блок «Мои анкеты услуг» со счётчиком `X из 3` и блокировкой занятых категорий при создании новой анкеты.
- Реализует публичную модалку **Профиль мастера** (`openMasterProfileModal`), модалку откликов с рейтингом и переходом в профиль (`openViewBidsModal`) и модалку отзывов (`openReviewFormModal`).
