const CATEGORY_LABELS = {
  electric: "Электрика",
  plumbing: "Сантехника",
  earthworks: "Земляные работы",
  repair: "Ремонт и отделка",
};

function escapeHtml(str) {
  return String(str ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

export function renderMasterCard(profile, currentUser) {
  const isOwner = currentUser && currentUser.id === profile.user_id;

  let actionsHtml = "";
  if (isOwner) {
    actionsHtml = `
      <div class="card-actions">
        <button class="btn btn-danger-outline btn-sm" data-action="delete-master" data-master-id="${profile.id}">
          Удалить анкету
        </button>
      </div>
    `;
  }

  return `
    <article class="card">
      <div>
        <div class="card-top">
          <span class="tag">${CATEGORY_LABELS[profile.category] || escapeHtml(profile.category)}</span>
          <span class="tag status-open">Опыт: ${profile.experience_years} лет</span>
        </div>
        <h3 class="card-title">${escapeHtml(profile.headline)}</h3>
        <p class="card-desc">${escapeHtml(profile.bio)}</p>
        ${profile.contact_phone ? `
          <div style="margin-top:10px; font-size:13px; color:var(--text-secondary);">
            📞 Телефон: <a href="tel:${escapeHtml(profile.contact_phone)}">${escapeHtml(profile.contact_phone)}</a>
          </div>
        ` : ""}
      </div>

      <div>
        <div class="card-meta">
          <span>📍 ${escapeHtml(profile.city || "Не указан")} • Мастер #${profile.user_id}</span>
          <span class="price">от ${Number(profile.min_price).toLocaleString("ru-RU")} ₽</span>
        </div>
        ${actionsHtml}
      </div>
    </article>
  `;
}
