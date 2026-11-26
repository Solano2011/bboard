import { CATEGORY_LABELS } from "../constants/categories.js";

const STATUS_LABELS = {
  open: "Открыта",
  in_progress: "В работе",
  pending_completion: "Ожидает подтверждения",
  completed: "Выполнена",
  canceled: "Отменена",
};

function escapeHtml(str) {
  return String(str ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
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
        <button class="btn btn-secondary btn-sm" data-action="edit-task" data-task-id="${task.id}">
          Редактировать
        </button>
        <button class="btn btn-danger-outline btn-sm" data-action="cancel-task" data-task-id="${task.id}">
          Отменить
        </button>
      `;
    } else if (!isGuest) {
      actionsHtml = `
        <button class="btn btn-primary btn-sm" data-action="open-bid" data-task-id="${task.id}" data-task-title="${escapeHtml(task.title)}">
          Откликнуться
        </button>
      `;
    }
  } else if (task.status === "in_progress") {
    if (task.is_assigned_to_me) {
      actionsHtml += `
        <button class="btn btn-success btn-sm" data-action="mark-completed" data-task-id="${task.id}">
          Отметить выполненным
        </button>
      `;
    }
    if (isAuthor) {
      actionsHtml += `
        <button class="btn btn-success btn-sm" data-action="complete-task" data-task-id="${task.id}">
          Завершить работу
        </button>
        <button class="btn btn-danger-outline btn-sm" data-action="cancel-task" data-task-id="${task.id}">
          Отменить
        </button>
      `;
    }
  } else if (task.status === "pending_completion" && isAuthor) {
    actionsHtml = `
      <button class="btn btn-success btn-sm" data-action="complete-task" data-task-id="${task.id}">
        Подтвердить выполнение
      </button>
      <button class="btn btn-danger-outline btn-sm" data-action="cancel-task" data-task-id="${task.id}">
        Отменить
      </button>
    `;
  } else if (task.status === "completed" && isAuthor) {
    if (task.my_review) {
      actionsHtml = `
        <button class="btn btn-secondary btn-sm" data-action="edit-review" data-task-id="${task.id}">
          Редактировать отзыв
        </button>
        <button class="btn btn-danger-outline btn-sm" data-action="delete-review" data-review-id="${task.my_review.id}">
          Удалить отзыв
        </button>
      `;
    } else {
      actionsHtml = `
        <button class="btn btn-primary btn-sm" data-action="create-review" data-task-id="${task.id}">
          Оставить отзыв
        </button>
      `;
    }
  }

  if (task.assigned_master_id) {
    actionsHtml += `
      <button class="btn btn-secondary btn-sm" data-action="view-master-profile" data-master-profile-id="${task.assigned_master_id}">
        Профиль исполнителя
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

  let reviewHtml = "";
  if (task.my_review) {
    reviewHtml = `
      <div class="task-review-box">
        <div><strong>Ваш отзыв:</strong> ★ ${Number(task.my_review.rating)}/5</div>
        ${task.my_review.comment ? `<div style="margin-top:4px;">${escapeHtml(task.my_review.comment)}</div>` : ""}
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
        ${reviewHtml}
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
