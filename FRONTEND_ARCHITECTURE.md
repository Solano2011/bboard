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
- Содержит семантические контейнеры (`#navActions`, `#primaryCta`, `#contentGrid`, `#modalBackdrop`), в которые JS-компоненты монтируют интерфейс.
- **Зависимости:** `web/css/styles.css`, `web/js/app.js` (используется в `internal/delivery/http/router.go`).

### `web/css/styles.css`
- **Паттерн:** Design Tokens (CSS Custom Properties) + BEM-like Component Classes.
- Реализует дизайн-систему **Flat Design 2.0** через CSS-переменные в `:root` (`--primary`, `--success`, `--warning`, `--danger`).
- Определяет пастельные бейджи статусов (`.status-open`, `.status-in_progress`, `.status-completed`, `.status-canceled`), адаптивную сетку карточек (`.grid` с микро-подъёмом при наведении), модальные окна (`.modal-backdrop`, `.modal-dialog` с `max-height` и скроллом) и уведомления (`.toast`).

---

## 2. Слой клиентской логики (`web/js/`)

### `web/js/api/httpClient.js` (Слой транспорта)
- **Паттерн:** Facade + Token Provider Injection (Dependency Inversion).
- Единый адаптер поверх `Fetch API` для запросов к `/api/v1/*` (UI-компоненты не вызывают `fetch` напрямую).
- Автоматически добавляет `Authorization: Bearer <token>`, получая токен через внедрённую функцию `tokenProvider` (без циклической зависимости со `Store`).
- Формирует `URLSearchParams` для фильтрации задач и мастеров по категории, городу и статусу, предоставляет метод `deleteMasterProfile(id)` для отправки `DELETE /api/v1/masters/{id}`.
- Перехватывает HTTP-ошибки (`!response.ok`) и выбрасывает единый класс `ApiError` с текстом ошибки от Go-бэкенда.

### `web/js/store/store.js` (Слой состояния)
- **Паттерн:** Observer (Pub-Sub) + Singleton + Web Storage Persistence.
- Единый источник истины (Single Source of Truth): хранит сессию пользователя, активную вкладку, фильтры и загруженные списки сущностей.
- `subscribe(listener)` регистрирует функции перерисовки UI, вызываемые при каждом обновлении через `setState(patch)`.
- `setSession(token, user)` и `clearSession()` синхронизируют JWT-авторизацию с `localStorage` (сессия сохраняется после перезагрузки страницы `F5`), а приватный метод `#loadSession()` безопасно восстанавливает её при старте.

### `web/js/components/` (Слой представления)
Изолированные рендер-функции, принимающие данные и возвращающие HTML-разметку без знания о сети и БД.
- **`taskCard.js` (`renderTaskCard`)**:
  - **Паттерн:** Role-Based UI Rendering + XSS Escaping.
  - Рендерит кнопки действий в зависимости от роли пользователя («Отклики и выбор мастера» для автора, «Откликнуться» для мастера) и скрывает их для статусов `completed` и `canceled`.
  - Отображает район (`district`) для всех пользователей и блок с телефоном и точным адресом выполнения только при их наличии в ответе сервера (для автора и назначенного исполнителя).
  - Экранирует пользовательский ввод через `escapeHtml()`.
- **`masterCard.js` (`renderMasterCard`)**:
  - **Паттерн:** Role-Based UI Rendering + Stateless Functional Component.
  - Рендерит карточку профиля мастера (специализация, `experience_years`, описание услуг, город, `min_price`, контактный телефон) с экранированием всех текстовых полей.
  - Для владельца анкеты (`currentUser.id === profile.user_id`) отображает кнопку «Удалить анкету» (`data-action="delete-master"`).
- **`modals.js` (`ModalManager` & `showToast`)**:
  - **Паттерн:** Singleton UI Controller + Notification Queue.
  - `ModalManager` управляет открытием/закрытием `#modalBackdrop` (по крестику и клику вне окна) и очисткой DOM.
  - `showToast(message, type)` создаёт самоуничтожающиеся уведомления (`success` / `error` / `info`) в правом нижнем углу экрана.

### `web/js/app.js` (Контроллер приложения)
- **Паттерн:** Mediator + Event Delegation (Делегирование событий).
- Главный координатор, связывающий `httpClient`, `Store` и UI-компоненты.
- Подписывается на `store.subscribe()` для автоматической перерисовки шапки (`renderNavbar`) и сетки (`renderGrid`).
- Использует глобальное делегирование событий (`document.addEventListener("click")`) для обработки кликов по динамически создаваемым кнопкам карточек и вкладок.
- Управляет сценариями модальных форм: вход, регистрация, создание заказа, создание анкеты мастера, подача отклика и выбор исполнителя, а также подтверждением и удалением собственной анкеты мастера с последующей перезагрузкой состояния.
  - В формах создания задачи и анкеты мастера предоставляет поля контактного телефона с клиентской нормализацией и строгой валидацией формата `+7XXXXXXXXXX`.
- **Зависимости:** `web/js/api/httpClient.js`, `web/js/store/store.js`, `web/js/components/taskCard.js`, `web/js/components/masterCard.js`, `web/js/components/modals.js`.
