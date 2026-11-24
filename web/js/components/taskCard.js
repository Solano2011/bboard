const CATEGORY_LABELS = {
  electric: "Электрика",
  plumbing: "Сантехника",
  earthworks: "Земляные работы",
  repair: "Ремонт и отделка",
};

const STATUS_LABELS = {
  open: "Открыта",
  in_progress: "В работе",
  completed: "Выполнена",
  canceled: "Отменена",
};

function escapeHtml(str) {
  return String(str ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

export function renderTaskCard(task, currentUser) {
  const isAuthor = currentUser && currentUser.id === task.author_id;
  const isGuest = !currentUser;

  let actionsHtml = "";

  if (task.status === "open") {
    if (isAuthor) {
      actionsHtml = `
        <button class="btn btn-primary btn-sm" data-action="view-bids" data-task-id="${task.id}">
          Отклики и выбор мастера
        </button>
        <button class="btn btn-danger-outline btn-sm" data-action="cancel-task" data-task-id="${task.id}">
          Отменить
        </button>
      `;
    } else if (!isGuest) {
      actionsHtml = `
        <button class="btn btn-primary btn-sm" data-action="open-bid" data-task-id="${task.id}" data-task-title="${escapeHtml(task.title)}">
          Откликнуться (предложить цену)
        </button>
      `;
    }
  } else if (task.status === "in_progress" && isAuthor) {
    actionsHtml = `
      <button class="btn btn-success btn-sm" data-action="complete-task" data-task-id="${task.id}">
        Завершить работу
      </button>
      <button class="btn btn-danger-outline btn-sm" data-action="cancel-task" data-task-id="${task.id}">
        Отменить
      </button>
    `;
  }

  let contactInfoHtml = "";
  if (task.contact_phone || task.work_address) {
    contactInfoHtml = `
      <div class="task-contacts" style="margin-top:10px; padding:8px 12px; background:var(--bg-secondary, #f8fafc); border:1px solid var(--border-color, #e2e8f0); border-radius:6px; font-size:13px;">
        <div style="font-weight:600; margin-bottom:4px; color:var(--text-main, #1e293b);">🔒 Контакты и адрес:</div>
        ${task.contact_phone ? `<div>📞 Телефон: <a href="tel:${escapeHtml(task.contact_phone)}">${escapeHtml(task.contact_phone)}</a></div>` : ""}
        ${task.work_address ? `<div>📍 Точный адрес: ${escapeHtml(task.work_address)}</div>` : ""}
      </div>
    `;
  }

  const locationText = task.district
    ? `${escapeHtml(task.city || "Город не указан")}, р-н ${escapeHtml(task.district)}`
    : escapeHtml(task.city || "Город не указан");

  return `
    <article class="card">
      <div>
        <div class="card-top">
          <span class="tag">${CATEGORY_LABELS[task.category] || escapeHtml(task.category)}</span>
          <span class="tag status-${task.status}">${STATUS_LABELS[task.status] || task.status}</span>
        </div>
        <h3 class="card-title">${escapeHtml(task.title)}</h3>
        <p class="card-desc">${escapeHtml(task.description)}</p>
        ${contactInfoHtml}
      </div>

      <div>
        <div class="card-meta">
          <span>📍 ${locationText} • ID #${task.id}</span>
          <span class="price">${Number(task.budget).toLocaleString("ru-RU")} ₽</span>
        </div>
        ${actionsHtml ? `<div class="card-actions">${actionsHtml}</div>` : ""}
      </div>
    </article>
  `;
}
