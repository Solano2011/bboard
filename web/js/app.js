import { api } from "./api/httpClient.js";
import { store } from "./store/store.js";
import { renderTaskCard } from "./components/taskCard.js";
import { renderMasterCard } from "./components/masterCard.js";
import { ModalManager, showToast } from "./components/modals.js";

const modal = new ModalManager();
api.setTokenProvider(() => store.state.token);

// --- Data Loaders ---
async function loadData() {
  const { activeTab, filters } = store.state;
  try {
    if (activeTab === "tasks") {
      const tasks = await api.listTasks(filters);
      store.setState({ tasks });
    } else {
      const masters = await api.listMasters({ category: filters.category, city: filters.city });
      store.setState({ masters });
    }
  } catch (err) {
    showToast(err.message, "error");
  }
}

// --- UI Renderers ---
function renderNavbar(state) {
  const navActions = document.getElementById("navActions");
  const primaryCta = document.getElementById("primaryCta");

  document.querySelectorAll(".nav-tab").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.tab === state.activeTab);
  });

  document.getElementById("filterStatus").style.display = state.activeTab === "tasks" ? "inline-block" : "none";

  if (state.user) {
    navActions.innerHTML = `
      <span class="user-pill">👤 ${state.user.name} (ID #${state.user.id})</span>
      <button class="btn btn-secondary btn-sm" id="btnLogout">Выйти</button>
    `;
    primaryCta.innerHTML =
      state.activeTab === "tasks"
        ? `<button class="btn btn-primary" id="btnOpenCreateTask">+ Разместить заказ</button>`
        : `<button class="btn btn-primary" id="btnOpenCreateMaster">+ Анкета мастера</button>`;
  } else {
    navActions.innerHTML = `
      <button class="btn btn-secondary btn-sm" id="btnOpenLogin">Войти</button>
      <button class="btn btn-primary btn-sm" id="btnOpenRegister">Регистрация</button>
    `;
    primaryCta.innerHTML = `<span style="font-size:13px; color:var(--text-muted)">Войдите, чтобы создать заказ или анкету</span>`;
  }
}

function renderGrid(state) {
  const grid = document.getElementById("contentGrid");

  if (state.activeTab === "tasks") {
    if (!state.tasks.length) {
      grid.innerHTML = `<div class="empty-state">Заказов по выбранным фильтрам пока нет</div>`;
      return;
    }
    grid.innerHTML = state.tasks.map((t) => renderTaskCard(t, state.user)).join("");
  } else {
    if (!state.masters.length) {
      grid.innerHTML = `<div class="empty-state">Анкет мастеров по выбранным фильтрам пока нет</div>`;
      return;
    }
    grid.innerHTML = state.masters.map((m) => renderMasterCard(m)).join("");
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
        <input class="input" type="email" name="email" value="ivan@example.com" required />
      </div>
      <div class="form-group">
        <label>Пароль</label>
        <input class="input" type="password" name="password" value="password123" required />
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
      <div class="form-group"><label>Пароль (от 6 символов)</label><input class="input" type="password" name="password" required /></div>
      <button class="btn btn-primary" style="width:100%" type="submit">Создать аккаунт</button>
    </form>
  `
  );

  document.getElementById("registerForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    try {
      await api.register(Object.fromEntries(fd.entries()));
      const res = await api.login(fd.get("email"), fd.get("password"));
      store.setSession(res.access_token, res.user);
      modal.close();
      showToast("Аккаунт создан!", "success");
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  });
}

function openCreateTaskModal() {
  modal.open(
    "Новое объявление заказчика",
    `
    <form id="createTaskForm">
      <div class="form-group"><label>Заголовок</label><input class="input" name="title" placeholder="Например: Выкопать яму под септик" required /></div>
      <div class="form-group">
        <label>Категория</label>
        <select class="input" name="category">
          <option value="electric">Электрика</option>
          <option value="plumbing">Сантехника</option>
          <option value="earthworks">Земляные работы</option>
          <option value="repair">Ремонт и отделка</option>
        </select>
      </div>
      <div class="form-group"><label>Город</label><input class="input" name="city" value="Астрахань" required /></div>
      <div class="form-group"><label>Район (необязательно)</label><input class="input" name="district" placeholder="Например: Советский" /></div>
      <div class="form-group">
        <label>Контактный телефон (необязательно)</label>
        <input class="input" name="contact_phone" placeholder="+79991234567" />
      </div>
      <div class="form-group">
        <label>Точный адрес (необязательно)</label>
        <input class="input" name="work_address" placeholder="Улица, дом, квартира" />
        <small style="color:var(--text-muted); font-size:12px; display:block; margin-top:4px;">
          Адрес и телефон увидит только выбранный мастер
        </small>
      </div>
      <div class="form-group"><label>Бюджет (₽)</label><input class="input" type="number" name="budget" value="3000" min="0" required /></div>
      <div class="form-group"><label>Подробное описание</label><textarea class="textarea" name="description" rows="3" required></textarea></div>
      <button class="btn btn-primary" style="width:100%" type="submit">Опубликовать</button>
    </form>
  `
  );

  document.getElementById("createTaskForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);

    let contactPhone = (fd.get("contact_phone") || "").trim();
    if (contactPhone) {
      let cleaned = contactPhone.replace(/[\s()\-]/g, "");
      if (cleaned.startsWith("8") && cleaned.length === 11) {
        cleaned = "+7" + cleaned.slice(1);
      }
      if (!/^\+7\d{10}$/.test(cleaned)) {
        showToast("Некорректный номер телефона. Формат: +7XXXXXXXXXX", "error");
        return;
      }
      contactPhone = cleaned;
    }

    try {
      await api.createTask({
        title: fd.get("title"),
        category: fd.get("category"),
        city: fd.get("city"),
        district: (fd.get("district") || "").trim(),
        contact_phone: contactPhone,
        work_address: (fd.get("work_address") || "").trim(),
        budget: Number(fd.get("budget")),
        description: fd.get("description"),
      });
      modal.close();
      showToast("Объявление опубликовано!", "success");
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  });
}

function openCreateMasterModal() {
  modal.open(
    "Создать анкету мастера",
    `
    <form id="createMasterForm">
      <div class="form-group"><label>Заголовок услуги</label><input class="input" name="headline" placeholder="Частный электрик, монтаж под ключ" required /></div>
      <div class="form-group">
        <label>Специализация</label>
        <select class="input" name="category">
          <option value="electric">Электрика</option>
          <option value="plumbing">Сантехника</option>
          <option value="earthworks">Земляные работы</option>
          <option value="repair">Ремонт и отделка</option>
        </select>
      </div>
      <div class="form-group"><label>Город</label><input class="input" name="city" value="Астрахань" required /></div>
      <div class="form-group"><label>Опыт работы (лет)</label><input class="input" type="number" name="experience_years" value="5" min="0" required /></div>
      <div class="form-group"><label>Минимальная цена выезда (₽)</label><input class="input" type="number" name="min_price" value="1500" min="0" required /></div>
      <div class="form-group"><label>О себе и инструменте</label><textarea class="textarea" name="bio" rows="3" required></textarea></div>
      <button class="btn btn-primary" style="width:100%" type="submit">Сохранить анкету</button>
    </form>
  `
  );

  document.getElementById("createMasterForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    try {
      await api.createMasterProfile({
        headline: fd.get("headline"),
        category: fd.get("category"),
        city: fd.get("city"),
        experience_years: Number(fd.get("experience_years")),
        min_price: Number(fd.get("min_price")),
        bio: fd.get("bio"),
      });
      modal.close();
      showToast("Анкета мастера создана!", "success");
      loadData();
    } catch (err) {
      showToast(err.message, "error");
    }
  });
}

function openBidModal(taskId, taskTitle) {
  modal.open(
    `Отклик на: ${taskTitle}`,
    `
    <form id="bidForm">
      <div class="form-group"><label>Ваша цена (₽)</label><input class="input" type="number" name="proposed_price" value="2500" min="0" required /></div>
      <div class="form-group"><label>Комментарий заказчику</label><textarea class="textarea" name="comment" rows="3" placeholder="Опишите сроки и наличие инструмента..." required></textarea></div>
      <button class="btn btn-primary" style="width:100%" type="submit">Отправить отклик</button>
    </form>
  `
  );

  document.getElementById("bidForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    try {
      await api.createBid(taskId, {
        proposed_price: Number(fd.get("proposed_price")),
        comment: fd.get("comment"),
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
    const listHtml = bids.length
      ? bids
          .map(
            (b) => `
          <div class="bid-item">
            <div class="bid-item-header">
              <span>Мастер #${b.master_id}</span>
              <span>${Number(b.proposed_price).toLocaleString("ru-RU")} ₽</span>
            </div>
            <p style="font-size:14px; color:var(--text-secondary); margin-bottom:10px;">${b.comment}</p>
            <button class="btn btn-primary btn-sm" data-assign-task="${taskId}" data-assign-master="${b.master_id}">
              Выбрать исполнителем
            </button>
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
    store.clearSession();
    showToast("Вы вышли из аккаунта");
    loadData();
  } else if (target.id === "btnOpenCreateTask") {
    openCreateTaskModal();
  } else if (target.id === "btnOpenCreateMaster") {
    openCreateMasterModal();
  } else if (target.id === "btnApplyFilters") {
    store.setState({
      filters: {
        category: document.getElementById("filterCategory").value,
        city: document.getElementById("filterCity").value,
        status: document.getElementById("filterStatus").value,
      },
    });
    loadData();
  } else if (target.dataset.action === "open-bid") {
    openBidModal(target.dataset.taskId, target.dataset.taskTitle);
  } else if (target.dataset.action === "view-bids") {
    openViewBidsModal(target.dataset.taskId);
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
  }
});

// Initial render & fetch
renderNavbar(store.state);
loadData();
