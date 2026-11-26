import { CATEGORY_LABELS } from "../constants/categories.js";

function escapeHtml(str) {
  return String(str ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

export function formatRating(ratingAvg, reviewsCount) {
  if (!reviewsCount) {
    return "Нет отзывов";
  }
  return `★ ${Number(ratingAvg).toFixed(1)} (${reviewsCount})`;
}

export function renderMasterCard(serviceItem) {
  const ratingText = formatRating(serviceItem.rating_avg, serviceItem.reviews_count);
  const locationParts = [serviceItem.master_city || "Город не указан"];
  if (serviceItem.master_district) {
    locationParts.push(`р-н ${serviceItem.master_district}`);
  }

  return `
    <article class="card">
      <div>
        <div class="card-top">
          <span class="tag">${CATEGORY_LABELS[serviceItem.category] || escapeHtml(serviceItem.category)}</span>
          <span class="tag rating-badge">${ratingText}</span>
        </div>
        <h3 class="card-title">${escapeHtml(serviceItem.title)}</h3>
        <p class="card-desc">${escapeHtml(serviceItem.experience_description || "Опыт не указан")}</p>
        <div style="margin-top:8px; font-size:13px; color:var(--text-secondary);">
          👤 ${escapeHtml(serviceItem.master_display_name)} • Выполнено заказов: ${Number(serviceItem.completed_orders_count || 0)}
        </div>
      </div>

      <div>
        <div class="card-meta">
          <span>📍 ${escapeHtml(locationParts.join(", "))}</span>
          <span class="price">от ${Number(serviceItem.price).toLocaleString("ru-RU")} ₽</span>
        </div>
        <div class="card-actions">
          <button class="btn btn-secondary btn-sm" data-action="view-master-profile" data-master-profile-id="${serviceItem.master_profile_id}">
            Профиль мастера
          </button>
        </div>
      </div>
    </article>
  `;
}
