import { api } from "./api/httpClient.js";
import { store } from "./store/store.js";
import { renderTaskCard } from "./components/taskCard.js";
import { renderMasterCard, formatRating } from "./components/masterCard.js";
import { ModalManager, showToast } from "./components/modals.js";
import { CATEGORIES, CATEGORY_LABELS, renderCategoryOptions } from "./constants/categories.js";

const modal = new ModalManager();
let lastOpenedProfileServices = [];

api.setTokenProvider(() => store.state.token);
api.setTokenUpdatedHandler((accessToken, user) => {
  store.setSession(accessToken, user || store.state.user);
});
api.setUnauthorizedHandler(() => {
  if (store.state.token) {
    const wasRestrictedTab =
      store.state.activeTab === "cabinet" || store.state.activeTab === "admin";
    store.clearSession();
    modal.close();
    showToast("Сессия истекла, войдите заново", "info");
    if (wasRestrictedTab) {
      loadData();
    }
  }
});

function escapeHtml(str) {
  return String(str ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function isModOrAdmin(user) {
  return Boolean(user && (user.role === "moderator" || user.role === "admin"));
}

function formatRoleBadge(role) {
  if (role === "admin") {
    return `<span class="role-badge role-admin">Администратор</span>`;
  }
  if (role === "moderator") {
    return `<span class="role-badge role-moderator">Модератор</span>`;
  }
  return `<span class="role-badge role-user">Пользователь</span>`;
}

function normalizePhone(raw, required = false) {
  const contactPhone = (raw || "").trim();
  if (!contactPhone) {
    if (required) {
      throw new Error("Укажите контактный телефон");
    }
    return "";
  }
  let cleaned = contactPhone.replace(/[\s()\-]/g, "");
  if (cleaned.startsWith("8") && cleaned.length === 11) {
    cleaned = "+7" + cleaned.slice(1);
  }
  if (!/^\+7\d{10}$/.test(cleaned)) {
    throw new Error("Некорректный номер телефона. Формат: +7XXXXXXXXXX");
  }
  return cleaned;
}

// --- Data Loaders ---
async function loadData() {
  const { activeTab, filters, user, adminSearch } = store.state;
  try {
    if (activeTab === "tasks") {
      const tasks = await api.listTasks(filters);
      store.setState({ tasks });
    } else if (activeTab === "masters") {
      const masters = await api.listMasterServices({ category: filters.category, city: filters.city });
      store.setState({ masters });
    } else if (activeTab === "cabinet") {
      if (!user) {
        store.setState({ activeTab: "tasks", myProfile: null, myServices: [] });
        return;
      }
      try {
        const res = await api.getMyMasterProfile();
        store.setState({ myProfile: res.profile, myServices: res.services || [] });
      } catch (err) {
        if (err.status === 404) {
          store.setState({ myProfile: null, myServices: [] });
        } else {
          throw err;
        }
      }
    } else if (activeTab === "admin") {
      if (!user || user.role !== "admin") {
        store.setState({ activeTab: "tasks", adminUsers: [] });
        return;
      }
      const users = await api.listAdminUsers({ search: adminSearch, limit: 50, offset: 0 });
      store.setState({ adminUsers: users });
    }
  } catch (err) {
    showToast(err.message, "error");
  }
}

// --- UI Renderers ---
function renderNavbar(state) {
  const navActions = document.getElementById("navActions");
  const primaryCta = document.getElementById("primaryCta");
  const navTabCabinet = document.getElementById("navTabCabinet");
  const navTabAdmin = document.getElementById("navTabAdmin");
  const filtersBar = document.querySelector(".filters");

  if (navTabCabinet) {
    navTabCabinet.style.display = state.user ? "inline-block" : "none";
  }
  if (navTabAdmin) {
    navTabAdmin.style.display = state.user?.role === "admin" ? "inline-block" : "none";
  }

  document.querySelectorAll(".nav-tab").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.tab === state.activeTab);
  });

  const hideFilters = state.activeTab === "cabinet" || state.activeTab === "admin";
  if (filtersBar) {
    filtersBar.style.display = hideFilters ? "none" : "flex";
  }
  document.getElementById("filterStatus").style.display = state.activeTab === "tasks" ? "inline-block" : "none";

  if (state.user) {
    const rolePill = state.user.role && state.user.role !== "user" ? ` ${formatRoleBadge(state.user.role)}` : "";
    navActions.innerHTML = `
      <span class="user-pill">👤 ${escapeHtml(state.user.name)} (ID #${state.user.id})${rolePill}</span>
      <button class="btn btn-secondary btn-sm" id="btnLogout">Выйти</button>
    `;
    if (state.activeTab === "tasks") {
      primaryCta.innerHTML = `<button class="btn btn-primary" id="btnOpenCreateTask">+ Разместить заказ</button>`;
    } else if (state.activeTab === "masters") {
      primaryCta.innerHTML = `<button class="btn btn-primary" id="btnGoCabinet">Мой кабинет мастера</button>`;
    } else {
      primaryCta.innerHTML = "";
    }
  } else {
    navActions.innerHTML = `
      <button class="btn btn-secondary btn-sm" id="btnOpenLogin">Войти</button>
      <button class="btn btn-primary btn-sm" id="btnOpenRegister">Регистрация</button>
    `;
    primaryCta.innerHTML = `<span style="font-size:13px; color:var(--text-muted)">Войдите, чтобы создать заказ или анкету</span>`;
  }
}

function renderCabinet(state) {
  const p = state.myProfile;
  const services = state.myServices || [];
  const activeCount = services.filter((s) => s.is_active).length;

  const servicesHtml = !p
    ? `<p style="color:var(--text-secondary); font-size:14px;">Сначала сохраните профиль мастера слева, чтобы добавлять анкеты услуг (до 3 активных).</p>`
    : services.length
    ? services
        .map(
          (s) => `
          <div class="service-item">
            <div class="card-top" style="margin-bottom:8px;">
              <span class="tag">${CATEGORY_LABELS[s.category] || escapeHtml(s.category)}</span>
              <span class="tag ${s.is_active ? "status-open" : "status-canceled"}">
                ${s.is_active ? "Активна" : "Скрыта"}
              </span>
            </div>
            <div style="font-weight:600; margin-bottom:4px;">${escapeHtml(s.title)}</div>
            <div style="font-size:13px; color:var(--text-secondary); margin-bottom:8px;">
              ${escapeHtml(s.experience_description || "Описание не указано")}
            </div>
            <div style="display:flex; justify-content:space-between; align-items:center; gap:8px;">
              <strong>от ${Number(s.price).toLocaleString("ru-RU")} ₽</strong>
              <div style="display:flex; gap:8px;">
                <button class="btn btn-secondary btn-sm" data-action="edit-service" data-service-id="${s.id}">Редактировать</button>
                <button class="btn btn-danger-outline btn-sm" data-action="delete-service" data-service-id="${s.id}">Удалить</button>
              </div>
            </div>
          </div>`
        )
        .join("")
    : `<p style="color:var(--text-secondary); font-size:14px;">У вас пока нет анкет услуг. Добавьте первую!</p>`;

  return `
    <div class="cabinet-grid">
      <section class="cabinet-panel">
        <div class="cabinet-panel-header">
          <h3>Профиль мастера</h3>
          ${
            p
              ? `<span class="tag rating-badge">${formatRating(p.rating_avg, p.reviews_count)} • Заказов: ${p.completed_orders_count}</span>`
              : `<span class="tag">Не создан</span>`
          }
        </div>
        <form id="cabinetProfileForm">
          <div class="form-group">
            <label>Отображаемое имя *</label>
            <input class="input" name="display_name" value="${escapeHtml(p?.display_name || state.user?.name || "")}" required />
          </div>
          <div class="form-group">
            <label>Ссылка на аватар (http/https, необязательно)</label>
            <input class="input" name="avatar_url" placeholder="https://..." value="${escapeHtml(p?.avatar_url || "")}" />
          </div>
          <div class="form-group">
            <label>Город *</label>
            <input class="input" name="city" value="${escapeHtml(p?.city || "Астрахань")}" required />
          </div>
          <div class="form-group">
            <label>Район (необязательно)</label>
            <input class="input" name="district" placeholder="Например: Советский" value="${escapeHtml(p?.district || "")}" />
          </div>
          <div class="form-group">
            <label>Телефон для связи *</label>
            <input class="input" name="contact_phone" placeholder="+79991234567" value="${escapeHtml(p?.contact_phone || state.user?.phone || "")}" required />
          </div>
          <div class="form-group">
            <label>Telegram (необязательно)</label>
            <input class="input" name="telegram" placeholder="@username" value="${escapeHtml(p?.telegram || "")}" />
          </div>
          <div class="form-group">
            <label>О себе</label>
            <textarea class="textarea" name="about" rows="3" placeholder="Расскажите об опыте и подходе к работе">${escapeHtml(p?.about || "")}</textarea>
          </div>
          <label class="switch-row">
            <input type="checkbox" name="is_contacts_public" ${p ? (p.is_contacts_public ? "checked" : "") : "checked"} />
            <span>Показывать мои контакты всем пользователям (иначе видны только после назначения на заказ)</span>
          </label>
          <button class="btn btn-primary" style="width:100%" type="submit">
            ${p ? "Сохранить профиль" : "Создать профиль мастера"}
          </button>
          ${
            p
              ? `<button class="btn btn-danger-outline" style="width:100%; margin-top:10px;" type="button" id="btnDeleteMyProfile">
                   Удалить анкету мастера
                 </button>`
              : ""
          }
        </form>
      </section>

      <section class="cabinet-panel">
        <div class="cabinet-panel-header">
          <h3>Мои анкеты услуг (${activeCount} из 3)</h3>
          ${
            p
              ? `<button class="btn btn-primary btn-sm" id="btnAddService" ${activeCount >= 3 ? "disabled" : ""}>+ Добавить услугу</button>`
              : ""
          }
        </div>
        ${servicesHtml}
      </section>
    </div>
  `;
}

function renderAdminUsers(state) {
  const users = state.adminUsers || [];
  const rowsHtml = users.length
    ? users
        .map((u) => {
          const isSelf = state.user && u.id === state.user.id;
          const isAdminUser = u.role === "admin";
          let actionHtml = `<span style="color:var(--text-muted); font-size:12px;">Недоступно</span>`;
          if (!isSelf && !isAdminUser) {
            actionHtml =
              u.role === "moderator"
                ? `<button class="btn btn-danger-outline btn-sm" data-action="set-user-role" data-user-id="${u.id}" data-role="user">Снять модератора</button>`
                : `<button class="btn btn-primary btn-sm" data-action="set-user-role" data-user-id="${u.id}" data-role="moderator">Назначить модератором</button>`;
          }

          return `
            <tr>
              <td>#${u.id}</td>
              <td>${escapeHtml(u.name)}</td>
              <td>${escapeHtml(u.email)}</td>
              <td>${formatRoleBadge(u.role)}</td>
              <td>${actionHtml}</td>
            </tr>
          `;
        })
        .join("")
    : `<tr><td colspan="5" style="text-align:center; color:var(--text-secondary);">Пользователи не найдены</td></tr>`;

  return `
    <section class="admin-panel">
      <div class="cabinet-panel-header">
        <h3>Управление пользователями и ролями</h3>
      </div>
      <form id="adminSearchForm" class="admin-search-row">
        <input class="input" style="flex:1; min-width:220px;" name="search" placeholder="Поиск по имени или email..." value="${escapeHtml(state.adminSearch || "")}" />
        <button class="btn btn-secondary" type="submit">Найти</button>
      </form>
      <div style="overflow-x:auto;">
        <table class="admin-users-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Имя</th>
              <th>Email</th>
              <th>Роль</th>
              <th>Действия</th>
            </tr>
          </thead>
          <tbody>
            ${rowsHtml}
          </tbody>
        </table>
      </div>
    </section>
  `;
}

function renderGrid(state) {
  const grid = document.getElementById("contentGrid");

  if (state.activeTab === "cabinet") {
    grid.innerHTML = renderCabinet(state);
    const profileForm = document.getElementById("cabinetProfileForm");
    if (profileForm) {
      profileForm.addEventListener("submit", async (e) => {
        e.preventDefault();
        const fd = new FormData(e.target);
        let contactPhone = "";
        try {
          contactPhone = normalizePhone(fd.get("contact_phone"), true);
        } catch (validationErr) {
          showToast(validationErr.message, "error");
          return;
        }

        try {
          await api.upsertMyMasterProfile({
            display_name: fd.get("display_name"),
            avatar_url: (fd.get("avatar_url") || "").trim(),
            city: fd.get("city"),
            district: (fd.get("district") || "").trim(),
            contact_phone: contactPhone,
            telegram: (fd.get("telegram") || "").trim(),
            about: fd.get("about"),
            is_contacts_public: fd.get("is_contacts_public") === "on",
          });
          showToast("Профиль мастера сохранён!", "success");
          loadData();
        } catch (err) {
          showToast(err.message, "error");
        }
      });
    }
    return;
  }

  if (state.activeTab === "admin") {
    grid.innerHTML = renderAdminUsers(state);
    const searchForm = document.getElementById("adminSearchForm");
    if (searchForm) {
      searchForm.addEventListener("submit", (e) => {
        e.preventDefault();
        const fd = new FormData(e.target);
        store.setState({ adminSearch: (fd.get("search") || "").trim() });
        loadData();
      });
    }
    return;
  }

  if (state.activeTab === "tasks") {
    if (!state.tasks.length) {
      grid.innerHTML = `<div class="empty-state">Заказов по выбранным фильтрам пока нет</div>`;
      return;
    }
    grid.innerHTML = state.tasks.map((t) => renderTaskCard(t, state.user)).join("");
  } else {
    if (!state.masters.length) {
      grid.innerHTML = `<div class="empty-state">Анкет услуг по выбранным фильтрам пока нет</div>`;
      return;
    }
    grid.innerHTML = state.masters.map((m) => renderMasterCard(m, state.user)).join("");
  }
}

store.subscribe((state) => {
  renderNavbar(state);
  renderGrid(state);
});

// --- Modals & Forms ---
function openLoginModal() {
  modal.open(
    "Вход в Bboard",
    `
    <form id="loginForm">
      <div class="form-group">
        <label>Email</label>
        <input class="input" type="email" name="email" placeholder="name@example.com" required />
      </div>
      <div class="form-group">
        <label>Пароль</label>
        <input class="input" type="password" name="password" placeholder="Введите пароль" required />
      </div>
      <button class="btn btn-primary" style="width:100%" type="submit">Войти</button>
    </form>
  `
  );

  document.getElementById("loginForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    try {
      const res = await api.login(fd.get("email"), fd.get("password"));
      store.setSession(res.access_token, res.user);
      modal.close();
      showToast(`Добро пожаловать, ${res.user.name}!`, "success");
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  });
}

function openRegisterModal() {
  modal.open(
    "Регистрация",
    `
    <form id="registerForm">
      <div class="form-group"><label>Имя</label><input class="input" name="name" required /></div>
      <div class="form-group"><label>Email</label><input class="input" type="email" name="email" required /></div>
      <div class="form-group"><label>Телефон</label><input class="input" name="phone" placeholder="+79990000000" /></div>
      <div class="form-group"><label>Пароль (от 8 символов)</label><input class="input" type="password" name="password" required /></div>
      <button class="btn btn-primary" style="width:100%" type="submit">Создать аккаунт</button>
    </form>
  `
  );

  document.getElementById("registerForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    try {
      const res = await api.register(Object.fromEntries(fd.entries()));
      store.setSession(res.access_token, res.user);
      modal.close();
      showToast("Аккаунт создан!", "success");
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  });
}

const MAX_TASK_PHOTOS = 3;
const MAX_TASK_PHOTO_BYTES = 5 * 1024 * 1024;
const ALLOWED_PHOTO_TYPES = new Set(["image/jpeg", "image/png", "image/webp"]);

function openTaskFormModal(existingTask = null) {
  const isEdit = Boolean(existingTask);
  const existingPhotos =
    isEdit && Array.isArray(existingTask.photos) ? [...existingTask.photos] : [];
  const markedForDeleteIds = new Set();
  const pendingFiles = []; // { file, objectUrl }

  const cleanupObjectUrls = () => {
    while (pendingFiles.length > 0) {
      const item = pendingFiles.pop();
      if (item?.objectUrl) {
        URL.revokeObjectURL(item.objectUrl);
      }
    }
  };

  modal.open(
    isEdit ? "Редактировать объявление" : "Новое объявление заказчика",
    `
    <form id="taskForm">
      <div class="form-group"><label>Заголовок</label><input class="input" name="title" placeholder="Например: Выкопать яму под септик" value="${isEdit ? escapeHtml(existingTask.title) : ""}" required /></div>
      <div class="form-group">
        <label>Категория</label>
        <select class="input" name="category">
          ${renderCategoryOptions(false, isEdit ? existingTask.category : "")}
        </select>
      </div>
      <div class="form-group"><label>Город</label><input class="input" name="city" value="${isEdit ? escapeHtml(existingTask.city || "") : "Астрахань"}" required /></div>
      <div class="form-group"><label>Район (необязательно)</label><input class="input" name="district" placeholder="Например: Советский" value="${isEdit ? escapeHtml(existingTask.district || "") : ""}" /></div>
      <div class="form-group">
        <label>Контактный телефон (необязательно)</label>
        <input class="input" name="contact_phone" placeholder="+79991234567" value="${isEdit ? escapeHtml(existingTask.contact_phone || "") : ""}" />
      </div>
      <div class="form-group">
        <label>Точный адрес (необязательно)</label>
        <input class="input" name="work_address" placeholder="Улица, дом, квартира" value="${isEdit ? escapeHtml(existingTask.work_address || "") : ""}" />
        <small style="color:var(--text-muted); font-size:12px; display:block; margin-top:4px;">
          Адрес и телефон увидит только выбранный мастер
        </small>
      </div>
      <div class="form-group"><label>Бюджет (₽)</label><input class="input" type="number" name="budget" value="${isEdit ? Number(existingTask.budget) : 3000}" min="0" required /></div>
      <div class="form-group"><label>Подробное описание</label><textarea class="textarea" name="description" rows="3" required>${isEdit ? escapeHtml(existingTask.description) : ""}</textarea></div>
      <div class="form-group">
        <label>Фотографии (<span id="taskPhotosCount">0</span> из ${MAX_TASK_PHOTOS})</label>
        <div class="photo-upload-box">
          <input class="input" type="file" id="taskPhotosInput" accept="image/jpeg,image/png,image/webp" multiple />
          <small style="color:var(--text-muted); font-size:12px; display:block; margin-top:4px;">
            До 3 фото (JPG, PNG, WebP, до 5 МБ каждое)
          </small>
          <div id="taskPhotoPreviews" class="photo-preview-grid"></div>
        </div>
      </div>
      <button class="btn btn-primary" style="width:100%" type="submit">${isEdit ? "Сохранить изменения" : "Опубликовать"}</button>
    </form>
  `,
    { onClose: cleanupObjectUrls }
  );

  const previewsContainer = document.getElementById("taskPhotoPreviews");
  const countEl = document.getElementById("taskPhotosCount");
  const fileInput = document.getElementById("taskPhotosInput");

  const getActiveExisting = () =>
    existingPhotos.filter((p) => !markedForDeleteIds.has(p.id));

  const renderPhotoPreviews = () => {
    const activeExisting = getActiveExisting();
    const total = activeExisting.length + pendingFiles.length;
    if (countEl) {
      countEl.textContent = String(total);
    }
    if (fileInput) {
      fileInput.disabled = total >= MAX_TASK_PHOTOS;
    }
    if (!previewsContainer) return;

    const existingHtml = activeExisting
      .map(
        (p) => `
        <div class="photo-preview-item">
          <img src="${escapeHtml(p.url)}" alt="Фото объявления" />
          <button type="button" class="photo-remove-btn" data-remove-existing-photo="${p.id}" title="Удалить фото">×</button>
        </div>`
      )
      .join("");

    const pendingHtml = pendingFiles
      .map(
        (item, idx) => `
        <div class="photo-preview-item">
          <img src="${escapeHtml(item.objectUrl)}" alt="Новое фото" />
          <button type="button" class="photo-remove-btn" data-remove-pending-photo="${idx}" title="Убрать фото">×</button>
        </div>`
      )
      .join("");

    previewsContainer.innerHTML = existingHtml + pendingHtml;
  };

  renderPhotoPreviews();

  if (fileInput) {
    fileInput.addEventListener("change", () => {
      const selected = Array.from(fileInput.files || []);
      fileInput.value = "";
      if (!selected.length) return;

      for (const file of selected) {
        const currentTotal = getActiveExisting().length + pendingFiles.length;
        if (currentTotal >= MAX_TASK_PHOTOS) {
          showToast(`Можно прикрепить не более ${MAX_TASK_PHOTOS} фотографий`, "error");
          break;
        }
        if (!ALLOWED_PHOTO_TYPES.has(file.type)) {
          showToast(`Неподдерживаемый формат файла "${file.name}". Разрешены JPG, PNG, WebP`, "error");
          continue;
        }
        if (file.size <= 0 || file.size > MAX_TASK_PHOTO_BYTES) {
          showToast(`Файл "${file.name}" превышает 5 МБ или пуст`, "error");
          continue;
        }
        pendingFiles.push({
          file,
          objectUrl: URL.createObjectURL(file),
        });
      }
      renderPhotoPreviews();
    });
  }

  if (previewsContainer) {
    previewsContainer.addEventListener("click", (e) => {
      const existingBtn = e.target.closest("[data-remove-existing-photo]");
      if (existingBtn) {
        markedForDeleteIds.add(Number(existingBtn.dataset.removeExistingPhoto));
        renderPhotoPreviews();
        return;
      }
      const pendingBtn = e.target.closest("[data-remove-pending-photo]");
      if (pendingBtn) {
        const idx = Number(pendingBtn.dataset.removePendingPhoto);
        const [removed] = pendingFiles.splice(idx, 1);
        if (removed?.objectUrl) {
          URL.revokeObjectURL(removed.objectUrl);
        }
        renderPhotoPreviews();
      }
    });
  }

  const taskFormEl = document.getElementById("taskForm");
  const submitBtn = taskFormEl?.querySelector('button[type="submit"]');
  const defaultSubmitText = submitBtn?.textContent || "";
  let isSubmitting = false;

  taskFormEl.addEventListener("submit", async (e) => {
    e.preventDefault();
    if (isSubmitting) return;

    const fd = new FormData(e.target);

    let contactPhone = "";
    try {
      contactPhone = normalizePhone(fd.get("contact_phone"), false);
    } catch (validationErr) {
      showToast(validationErr.message, "error");
      return;
    }

    const payload = {
      title: fd.get("title"),
      category: fd.get("category"),
      city: fd.get("city"),
      district: (fd.get("district") || "").trim(),
      contact_phone: contactPhone,
      work_address: (fd.get("work_address") || "").trim(),
      budget: Number(fd.get("budget")),
      description: fd.get("description"),
    };

    isSubmitting = true;
    if (submitBtn) {
      submitBtn.disabled = true;
      submitBtn.textContent = "Сохранение...";
    }

    try {
      if (isEdit) {
        await api.updateTask(existingTask.id, payload);
        let deletedAny = false;
        try {
          for (const photoId of markedForDeleteIds) {
            await api.deleteTaskPhoto(existingTask.id, photoId);
            deletedAny = true;
          }
          if (pendingFiles.length > 0) {
            await api.uploadTaskPhotos(
              existingTask.id,
              pendingFiles.map((item) => item.file)
            );
          }
          showToast("Объявление обновлено!", "success");
        } catch (photoErr) {
          if (deletedAny) {
            await loadData();
          }
          showToast(`Объявление сохранено, но ошибка с фото: ${photoErr.message}`, "error");
          modal.close();
          if (!deletedAny) {
            loadData();
          }
          return;
        }
      } else {
        const created = await api.createTask(payload);
        if (pendingFiles.length > 0) {
          try {
            await api.uploadTaskPhotos(
              created.id,
              pendingFiles.map((item) => item.file)
            );
            showToast("Объявление и фотографии опубликованы!", "success");
          } catch (uploadErr) {
            showToast(
              `Объявление создано, но фото не загрузились: ${uploadErr.message}. Их можно добавить через редактирование.`,
              "error"
            );
            modal.close();
            loadData();
            return;
          }
        } else {
          showToast("Объявление опубликовано!", "success");
        }
      }
      modal.close();
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    } finally {
      isSubmitting = false;
      if (submitBtn) {
        submitBtn.disabled = false;
        submitBtn.textContent = defaultSubmitText;
      }
    }
  });
}

function openTaskPhotosModal(taskId) {
  const task = store.state.tasks.find((item) => item.id === Number(taskId));
  if (!task || !Array.isArray(task.photos) || task.photos.length === 0) {
    return;
  }

  let currentIndex = 0;
  const photos = task.photos;

  const renderGallery = () => {
    const active = photos[currentIndex];
    const controlsHtml =
      photos.length > 1
        ? `
          <div class="gallery-controls">
            <button type="button" class="btn btn-secondary btn-sm" id="galleryPrevBtn">← Предыдущее</button>
            <span style="font-size:13px; color:var(--text-secondary);">Фото ${currentIndex + 1} из ${photos.length}</span>
            <button type="button" class="btn btn-secondary btn-sm" id="galleryNextBtn">Следующее →</button>
          </div>
          <div class="gallery-thumbs">
            ${photos
              .map(
                (p, idx) => `
                <button type="button" class="gallery-thumb${idx === currentIndex ? " active" : ""}" data-gallery-idx="${idx}">
                  <img src="${escapeHtml(p.url)}" alt="Миниатюра ${idx + 1}" />
                </button>`
              )
              .join("")}
          </div>
        `
        : `<div style="margin-top:8px; font-size:13px; color:var(--text-secondary);">Фото 1 из 1</div>`;

    modal.open(
      task.title,
      `
      <div class="gallery-stage">
        <img class="gallery-img" src="${escapeHtml(active.url)}" alt="${escapeHtml(task.title)}" />
        ${controlsHtml}
      </div>
    `
    );

    const prevBtn = document.getElementById("galleryPrevBtn");
    const nextBtn = document.getElementById("galleryNextBtn");
    if (prevBtn) {
      prevBtn.addEventListener("click", () => {
        currentIndex = (currentIndex - 1 + photos.length) % photos.length;
        renderGallery();
      });
    }
    if (nextBtn) {
      nextBtn.addEventListener("click", () => {
        currentIndex = (currentIndex + 1) % photos.length;
        renderGallery();
      });
    }
    document.querySelectorAll("[data-gallery-idx]").forEach((thumbBtn) => {
      thumbBtn.addEventListener("click", () => {
        currentIndex = Number(thumbBtn.dataset.galleryIdx);
        renderGallery();
      });
    });
  };

  renderGallery();
}

function openServiceFormModal(existingService = null, { isModeration = false, onSuccess = null } = {}) {
  const isEdit = Boolean(existingService);
  const takenCategories = isModeration
    ? new Set()
    : new Set(
        (store.state.myServices || [])
          .filter((s) => s.is_active && (!isEdit || s.id !== existingService.id))
          .map((s) => s.category)
      );

  const selectedCategory = isEdit
    ? existingService.category
    : CATEGORIES.find((c) => !takenCategories.has(c.value))?.value || "";

  const categoryOptionsHtml = CATEGORIES.map(({ value, label }) => {
    const isTaken = takenCategories.has(value);
    const isSelected = value === selectedCategory;
    const isDisabled = isTaken && !(isEdit && isSelected);
    return `<option value="${value}"${isSelected ? " selected" : ""}${isDisabled ? " disabled" : ""}>${label}${isTaken ? " (уже есть активная)" : ""}</option>`;
  }).join("");

  modal.open(
    isEdit ? "Редактировать анкету услуги" : "Новая анкета услуги",
    `
    <form id="serviceForm">
      <div class="form-group">
        <label>Категория *</label>
        <select class="input" name="category" required>
          ${categoryOptionsHtml}
        </select>
      </div>
      <div class="form-group">
        <label>Заголовок услуги *</label>
        <input class="input" name="title" placeholder="Например: Электромонтаж под ключ" value="${isEdit ? escapeHtml(existingService.title) : ""}" required />
      </div>
      <div class="form-group">
        <label>Минимальная цена / ставка (₽) *</label>
        <input class="input" type="number" name="price" value="${isEdit ? Number(existingService.price) : 1500}" min="0" required />
      </div>
      <div class="form-group">
        <label>Описание опыта в этой сфере</label>
        <textarea class="textarea" name="experience_description" rows="3" placeholder="Опыт работы, инструмент, гарантия...">${isEdit ? escapeHtml(existingService.experience_description || "") : ""}</textarea>
      </div>
      ${
        isEdit
          ? `<label class="switch-row">
               <input type="checkbox" name="is_active" ${existingService.is_active ? "checked" : ""} />
               <span>Анкета активна и видна в каталоге</span>
             </label>`
          : ""
      }
      <button class="btn btn-primary" style="width:100%" type="submit">${isEdit ? "Сохранить изменения" : "Создать услугу"}</button>
    </form>
  `
  );

  document.getElementById("serviceForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);

    const payload = {
      category: fd.get("category"),
      title: fd.get("title"),
      price: Number(fd.get("price")),
      experience_description: fd.get("experience_description") || "",
    };
    if (isEdit) {
      payload.is_active = fd.get("is_active") === "on";
    }

    try {
      if (isEdit) {
        await api.updateMasterService(existingService.id, payload);
        showToast("Анкета услуги обновлена!", "success");
      } else {
        await api.createMasterService(payload);
        showToast("Анкета услуги добавлена!", "success");
      }
      if (onSuccess) {
        await onSuccess();
      } else {
        modal.close();
      }
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  });
}

async function openMasterProfileModal(profileId) {
  try {
    const data = await api.getPublicMasterProfile(profileId);
    const p = data.profile;
    const services = data.services || [];
    const reviews = data.reviews || [];
    lastOpenedProfileServices = services;

    const modAccess = isModOrAdmin(store.state.user);

    const avatarHtml = p.avatar_url
      ? `<img class="profile-avatar" src="${escapeHtml(p.avatar_url)}" alt="${escapeHtml(p.display_name)}" />`
      : `<div class="profile-avatar">${escapeHtml((p.display_name || "М").slice(0, 1).toUpperCase())}</div>`;

    const contactsHtml = data.contacts_visible
      ? `
        <div class="task-contacts" style="margin-bottom:14px; padding:10px 12px; background:var(--bg-subtle, #f8fafc); border:1px solid var(--border, #e2e8f0); border-radius:8px; font-size:13px;">
          <div><strong>Контакты мастера:</strong></div>
          ${p.contact_phone ? `<div>📞 <a href="tel:${escapeHtml(p.contact_phone)}">${escapeHtml(p.contact_phone)}</a></div>` : ""}
          ${p.telegram ? `<div>✈️ Telegram: ${escapeHtml(p.telegram)}</div>` : ""}
        </div>
      `
      : `
        <div class="task-contacts" style="margin-bottom:14px; padding:10px 12px; background:var(--bg-subtle, #f8fafc); border:1px solid var(--border, #e2e8f0); border-radius:8px; font-size:13px; color:var(--text-secondary);">
          🔒 Контакты скрыты владельцем и откроются после назначения на заказ
        </div>
      `;

    const servicesHtml = services.length
      ? services
          .map(
            (s) => `
          <div class="service-item">
            <div class="card-top" style="margin-bottom:6px;">
              <span class="tag">${CATEGORY_LABELS[s.category] || escapeHtml(s.category)}</span>
              <strong>от ${Number(s.price).toLocaleString("ru-RU")} ₽</strong>
            </div>
            <div style="font-weight:600; margin-bottom:4px;">${escapeHtml(s.title)}</div>
            <div style="font-size:13px; color:var(--text-secondary); margin-bottom:8px;">${escapeHtml(s.experience_description || "")}</div>
            ${
              modAccess
                ? `<div style="display:flex; gap:8px;">
                     <button class="btn btn-secondary btn-sm" data-action="edit-profile-service" data-service-id="${s.id}" data-profile-id="${profileId}">Редактировать</button>
                     <button class="btn btn-danger-outline btn-sm" data-action="delete-profile-service" data-service-id="${s.id}" data-profile-id="${profileId}">Удалить</button>
                   </div>`
                : ""
            }
          </div>`
          )
          .join("")
      : `<p style="font-size:13px; color:var(--text-secondary);">Активных услуг пока нет.</p>`;

    const reviewsHtml = reviews.length
      ? reviews
          .map(
            (rev) => `
          <div class="review-item">
            <div style="display:flex; justify-content:space-between; margin-bottom:4px; font-size:13px;">
              <strong>${escapeHtml(rev.author_name || `Заказчик #${rev.author_user_id}`)}</strong>
              <span class="tag rating-badge">★ ${Number(rev.rating)}/5</span>
            </div>
            ${rev.comment ? `<div style="font-size:13px; color:var(--text-secondary); margin-bottom:6px;">${escapeHtml(rev.comment)}</div>` : ""}
            ${
              modAccess
                ? `<button class="btn btn-danger-outline btn-sm" data-action="delete-review" data-review-id="${rev.id}" data-profile-id="${profileId}">
                     Удалить отзыв
                   </button>`
                : ""
            }
          </div>`
          )
          .join("")
      : `<p style="font-size:13px; color:var(--text-secondary);">Отзывов пока нет.</p>`;

    modal.open(
      "Профиль мастера",
      `
      <div class="profile-header">
        ${avatarHtml}
        <div>
          <h3 style="margin:0 0 4px;">${escapeHtml(p.display_name)}</h3>
          <div style="font-size:13px; color:var(--text-secondary);">
            📍 ${escapeHtml(p.city || "Город не указан")}${p.district ? `, р-н ${escapeHtml(p.district)}` : ""}
          </div>
        </div>
      </div>

      <div class="profile-stats">
        <span class="tag rating-badge">${formatRating(p.rating_avg, p.reviews_count)}</span>
        <span class="tag status-completed">Выполнено заказов: ${Number(p.completed_orders_count || 0)}</span>
      </div>

      ${p.about ? `<p style="font-size:14px; color:var(--text-secondary); margin-bottom:14px;">${escapeHtml(p.about)}</p>` : ""}
      ${contactsHtml}

      <h4 style="margin:14px 0 8px;">Услуги мастера</h4>
      ${servicesHtml}

      <h4 style="margin:14px 0 8px;">Отзывы заказчиков (${reviews.length})</h4>
      ${reviewsHtml}
    `
    );
  } catch (err) {
    showToast(err.message, "error");
  }
}

function openBidModal(taskId, taskTitle) {
  modal.open(
    `Отклик на: ${taskTitle}`,
    `
    <form id="bidForm">
      <div class="form-group"><label>Ваша цена (₽, необязательно)</label><input class="input" type="number" name="proposed_price" placeholder="Например: 2500" min="0" /></div>
      <div class="form-group"><label>Комментарий заказчику (необязательно)</label><textarea class="textarea" name="comment" rows="3" placeholder="Опишите сроки и наличие инструмента..."></textarea></div>
      <button class="btn btn-primary" style="width:100%" type="submit">Отправить отклик</button>
    </form>
  `
  );

  document.getElementById("bidForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const rawPrice = (fd.get("proposed_price") || "").trim();
    try {
      await api.createBid(taskId, {
        proposed_price: rawPrice !== "" ? Number(rawPrice) : null,
        comment: (fd.get("comment") || "").trim(),
      });
      modal.close();
      showToast("Отклик отправлен заказчику!", "success");
    } catch (err) {
      showToast(err.message, "error");
    }
  });
}

async function openViewBidsModal(taskId) {
  try {
    const bids = await api.listTaskBids(taskId);
    const task = store.state.tasks.find((item) => item.id === Number(taskId));
    const canAssign = Boolean(
      task && store.state.user && task.author_id === store.state.user.id && task.status === "open"
    );

    const listHtml = bids.length
      ? bids
          .map(
            (b) => `
          <div class="bid-item">
            <div class="bid-item-header">
              <span><strong>${escapeHtml(b.master_display_name || `Мастер #${b.master_profile_id}`)}</strong></span>
              <span>${b.proposed_price !== undefined && b.proposed_price !== null ? `${Number(b.proposed_price).toLocaleString("ru-RU")} ₽` : "Цена по договорённости"}</span>
            </div>
            <div style="font-size:12px; color:var(--text-secondary); margin-bottom:6px;">
              ${formatRating(b.master_rating_avg, b.master_reviews_count)} • Выполнено заказов: ${Number(b.master_completed_orders_count || 0)}
            </div>
            ${b.comment ? `<p style="font-size:14px; color:var(--text-secondary); margin-bottom:10px;">${escapeHtml(b.comment)}</p>` : ""}
            <div style="display:flex; gap:8px; flex-wrap:wrap;">
              <button class="btn btn-secondary btn-sm" data-action="view-master-profile" data-master-profile-id="${b.master_profile_id}">
                Профиль мастера
              </button>
              ${
                canAssign
                  ? `<button class="btn btn-primary btn-sm" data-assign-task="${taskId}" data-assign-master="${b.master_profile_id}">
                       Выбрать исполнителем
                     </button>`
                  : ""
              }
            </div>
          </div>`
          )
          .join("")
      : `<p style="color:var(--text-secondary)">Пока никто из мастеров не откликнулся на это объявление.</p>`;

    modal.open(`Отклики на задачу #${taskId}`, listHtml);

    document.querySelectorAll("[data-assign-master]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        try {
          await api.assignMaster(btn.dataset.assignTask, btn.dataset.assignMaster);
          modal.close();
          showToast("Мастер назначен! Задача переведена в статус «В работе».", "success");
          loadData();
        } catch (err) {
          showToast(err.message, "error");
        }
      });
    });
  } catch (err) {
    showToast(err.message, "error");
  }
}

function openReviewFormModal(taskId, existingReview = null) {
  const isEdit = Boolean(existingReview);
  const currentRating = isEdit ? Number(existingReview.rating) : 5;
  const ratingOptions = [5, 4, 3, 2, 1]
    .map((val) => `<option value="${val}"${val === currentRating ? " selected" : ""}>${"★".repeat(val)} (${val})</option>`)
    .join("");

  modal.open(
    isEdit ? "Редактировать отзыв" : "Оставить отзыв о мастере",
    `
    <form id="reviewForm">
      <div class="form-group">
        <label>Оценка *</label>
        <select class="input" name="rating" required>
          ${ratingOptions}
        </select>
      </div>
      <div class="form-group">
        <label>Комментарий</label>
        <textarea class="textarea" name="comment" rows="3" placeholder="Поделитесь впечатлением о выполненной работе">${isEdit ? escapeHtml(existingReview.comment || "") : ""}</textarea>
      </div>
      <button class="btn btn-primary" style="width:100%" type="submit">${isEdit ? "Сохранить отзыв" : "Опубликовать отзыв"}</button>
    </form>
  `
  );

  document.getElementById("reviewForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const payload = {
      rating: Number(fd.get("rating")),
      comment: (fd.get("comment") || "").trim(),
    };

    try {
      if (isEdit) {
        await api.updateReview(existingReview.id, payload);
        showToast("Отзыв обновлён!", "success");
      } else {
        await api.createReview(taskId, payload);
        showToast("Отзыв опубликован!", "success");
      }
      modal.close();
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  });
}

// --- Global Event Delegation ---
document.addEventListener("click", async (e) => {
  const target = e.target.closest("button");
  if (!target) return;

  if (target.dataset.tab) {
    store.setState({ activeTab: target.dataset.tab });
    loadData();
  } else if (target.id === "btnOpenLogin") {
    openLoginModal();
  } else if (target.id === "btnOpenRegister") {
    openRegisterModal();
  } else if (target.id === "btnLogout") {
    try {
      await api.logout();
    } catch {
      // Локальную сессию сбрасываем даже при сетевой ошибке
    } finally {
      store.clearSession();
      showToast("Вы вышли из аккаунта");
      loadData();
    }
  } else if (target.id === "btnOpenCreateTask") {
    openTaskFormModal();
  } else if (target.id === "btnGoCabinet") {
    store.setState({ activeTab: "cabinet" });
    loadData();
  } else if (target.id === "btnAddService") {
    openServiceFormModal();
  } else if (target.id === "btnDeleteMyProfile") {
    if (!confirm("Вы уверены, что хотите удалить свою анкету мастера?")) {
      return;
    }
    try {
      await api.deleteMyMasterProfile();
      showToast("Анкета мастера удалена", "success");
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  } else if (target.id === "btnApplyFilters") {
    store.setState({
      filters: {
        category: document.getElementById("filterCategory").value,
        city: document.getElementById("filterCity").value,
        status: document.getElementById("filterStatus").value,
      },
    });
    loadData();
  } else if (target.dataset.action === "view-master-profile") {
    openMasterProfileModal(target.dataset.masterProfileId);
  } else if (target.dataset.action === "edit-service") {
    const svcId = Number(target.dataset.serviceId);
    const svc = (store.state.myServices || []).find((item) => item.id === svcId);
    if (svc) {
      openServiceFormModal(svc);
    }
  } else if (target.dataset.action === "delete-service") {
    if (!confirm("Удалить эту анкету услуги?")) {
      return;
    }
    try {
      await api.deleteMasterService(target.dataset.serviceId);
      showToast("Анкета услуги удалена", "success");
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  } else if (target.dataset.action === "edit-any-service") {
    const svcId = Number(target.dataset.serviceId);
    const svc = (store.state.masters || []).find((item) => item.id === svcId);
    if (svc) {
      openServiceFormModal(svc, { isModeration: true });
    }
  } else if (target.dataset.action === "delete-any-service") {
    if (!confirm("Удалить эту анкету услуги мастера?")) {
      return;
    }
    try {
      await api.deleteMasterService(target.dataset.serviceId);
      showToast("Анкета услуги удалена", "success");
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  } else if (target.dataset.action === "edit-profile-service") {
    const svcId = Number(target.dataset.serviceId);
    const profileId = target.dataset.profileId;
    const svc = lastOpenedProfileServices.find((item) => item.id === svcId);
    if (svc) {
      openServiceFormModal(svc, {
        isModeration: true,
        onSuccess: async () => {
          await openMasterProfileModal(profileId);
        },
      });
    }
  } else if (target.dataset.action === "delete-profile-service") {
    if (!confirm("Удалить эту анкету услуги мастера?")) {
      return;
    }
    const profileId = target.dataset.profileId;
    try {
      await api.deleteMasterService(target.dataset.serviceId);
      showToast("Анкета услуги удалена", "success");
      await openMasterProfileModal(profileId);
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  } else if (target.dataset.action === "view-task-photos") {
    openTaskPhotosModal(target.dataset.taskId);
  } else if (target.dataset.action === "open-bid") {
    openBidModal(target.dataset.taskId, target.dataset.taskTitle);
  } else if (target.dataset.action === "view-bids") {
    openViewBidsModal(target.dataset.taskId);
  } else if (target.dataset.action === "edit-task") {
    const taskId = Number(target.dataset.taskId);
    const task = store.state.tasks.find((item) => item.id === taskId);
    if (task) {
      openTaskFormModal(task);
    }
  } else if (target.dataset.action === "delete-task") {
    const taskId = Number(target.dataset.taskId);
    const task = store.state.tasks.find((item) => item.id === taskId);
    const isModDelete = task && store.state.user && task.author_id !== store.state.user.id;
    const msg = isModDelete
      ? "Вы удаляете чужое объявление как модератор. Это действие необратимо. Продолжить?"
      : "Вы уверены, что хотите удалить это объявление?";
    if (!confirm(msg)) {
      return;
    }
    try {
      await api.deleteTask(taskId);
      showToast("Объявление удалено", "success");
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  } else if (target.dataset.action === "mark-completed") {
    try {
      await api.markTaskCompleted(target.dataset.taskId);
      showToast("Задача отмечена выполненной! Ожидается подтверждение заказчика.", "success");
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  } else if (target.dataset.action === "complete-task") {
    try {
      await api.completeTask(target.dataset.taskId);
      showToast("Задача успешно завершена!", "success");
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  } else if (target.dataset.action === "cancel-task") {
    try {
      await api.cancelTask(target.dataset.taskId);
      showToast("Задача отменена");
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  } else if (target.dataset.action === "create-review") {
    openReviewFormModal(Number(target.dataset.taskId));
  } else if (target.dataset.action === "edit-review") {
    const taskId = Number(target.dataset.taskId);
    const task = store.state.tasks.find((item) => item.id === taskId);
    if (task?.my_review) {
      openReviewFormModal(taskId, task.my_review);
    }
  } else if (target.dataset.action === "delete-review") {
    if (!confirm("Удалить этот отзыв?")) {
      return;
    }
    const profileId = target.dataset.profileId;
    try {
      await api.deleteReview(target.dataset.reviewId);
      showToast("Отзыв удалён");
      if (profileId) {
        await openMasterProfileModal(profileId);
      }
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  } else if (target.dataset.action === "set-user-role") {
    const userId = target.dataset.userId;
    const role = target.dataset.role;
    const confirmText =
      role === "moderator"
        ? "Назначить этого пользователя модератором?"
        : "Снять права модератора с этого пользователя?";
    if (!confirm(confirmText)) {
      return;
    }
    try {
      await api.updateUserRole(userId, role);
      showToast("Роль пользователя обновлена", "success");
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  }
});

const filterCategorySelect = document.getElementById("filterCategory");
if (filterCategorySelect) {
  filterCategorySelect.innerHTML = renderCategoryOptions(true);
}

async function validateSessionOnStartup() {
  if (!store.state.token) return;
  try {
    const meUser = await api.me();
    if (meUser && meUser.id) {
      store.setSession(store.state.token, meUser);
    }
  } catch {
    // При протухшем refresh_token сессия уже сброшена в unauthorizedHandler
  }
}

// Initial render & fetch
(async function initApp() {
  renderNavbar(store.state);
  await validateSessionOnStartup();
  await loadData();
})();
