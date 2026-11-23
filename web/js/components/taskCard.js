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
    .replaceAll("&", "&")
    .replaceAll("<", "<")
    .replaceAll(">", ">");
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

  return `
    <article class="card">
      <div>
        <div class="card-top">
          <span class="tag">${CATEGORY_LABELS[task.category] || escapeHtml(task.category)}</span>
          <span class="tag status-${task.status}">${STATUS_LABELS[task.status] || task.status}</span>
        </div>
        <h3 class="card-title">${escapeHtml(task.title)}</h3>
        <p class="card-desc">${escapeHtml(task.description)}</p>
      </div>

      <div>
        <div class="card-meta">
          <span>📍 ${escapeHtml(task.city || "Город не указан")} • ID #${task.id}</span>
          <span class="price">${Number(task.budget).toLocaleString("ru-RU")} ₽</span>
        </div>
        ${actionsHtml ? `<div class="card-actions">${actionsHtml}</div>` : ""}
      </div>
    </article>
  `;
}