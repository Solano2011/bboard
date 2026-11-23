export class ModalManager {
  constructor() {
    this.backdrop = document.getElementById("modalBackdrop");
    this.titleEl = document.getElementById("modalTitle");
    this.bodyEl = document.getElementById("modalBody");
    document.getElementById("modalClose").addEventListener("click", () => this.close());
    this.backdrop.addEventListener("click", (e) => {
      if (e.target === this.backdrop) this.close();
    });
  }

  open(title, htmlContent) {
    this.titleEl.textContent = title;
    this.bodyEl.innerHTML = htmlContent;
    this.backdrop.classList.remove("hidden");
  }

  close() {
    this.backdrop.classList.add("hidden");
    this.bodyEl.innerHTML = "";
  }
}

export function showToast(message, type = "info") {
  const container = document.getElementById("toastContainer");
  const el = document.createElement("div");
  el.className = `toast ${type}`;
  el.textContent = message;
  container.appendChild(el);
  setTimeout(() => el.remove(), 3500);
}