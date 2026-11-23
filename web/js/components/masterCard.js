const CATEGORY_LABELS = {
  electric: "Электрика",
  plumbing: "Сантехника",
  earthworks: "Земляные работы",
  repair: "Ремонт и отделка",
};

function escapeHtml(str) {
  return String(str ?? "")
    .replaceAll("&", "&")
    .replaceAll("<", "<")
    .replaceAll(">", ">");
}

export function renderMasterCard(profile) {
  return `
    <article class="card">
      <div>
        <div class="card-top">
          <span class="tag">${CATEGORY_LABELS[profile.category] || escapeHtml(profile.category)}</span>
          <span class="tag status-open">Опыт: ${profile.experience_years} лет</span>
        </div>
        <h3 class="card-title">${escapeHtml(profile.headline)}</h3>
        <p class="card-desc">${escapeHtml(profile.bio)}</p>
      </div>

      <div>
        <div class="card-meta">
          <span>📍 ${escapeHtml(profile.city || "Не указан")} • Мастер #${profile.user_id}</span>
          <span class="price">от ${Number(profile.min_price).toLocaleString("ru-RU")} ₽</span>
        </div>
      </div>
    </article>
  `;
}