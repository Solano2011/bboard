export class ApiError extends Error {
  constructor(status, message) {
    super(message);
    this.status = status;
  }
}

class HttpClient {
  constructor(baseUrl = "/api/v1") {
    this.baseUrl = baseUrl;
    this.tokenProvider = () => null;
    this.tokenUpdatedHandler = () => {};
    this.unauthorizedHandler = () => {};
    this.refreshPromise = null;
  }

  setTokenProvider(fn) {
    this.tokenProvider = fn;
  }

  setTokenUpdatedHandler(fn) {
    this.tokenUpdatedHandler = fn;
  }

  setUnauthorizedHandler(fn) {
    this.unauthorizedHandler = fn;
  }

  #isAuthEndpoint(path) {
    return (
      path.startsWith("/auth/login") ||
      path.startsWith("/auth/register") ||
      path.startsWith("/auth/refresh") ||
      path.startsWith("/auth/logout")
    );
  }

  async #refreshAccessToken() {
    if (!this.refreshPromise) {
      this.refreshPromise = (async () => {
        const response = await fetch(`${this.baseUrl}/auth/refresh`, {
          method: "POST",
          credentials: "include",
        });
        const data = await response.json().catch(() => ({}));
        if (!response.ok || !data.access_token) {
          throw new ApiError(response.status, data.error || "Сессия истекла");
        }
        this.tokenUpdatedHandler(data.access_token, data.user);
        return data.access_token;
      })().finally(() => {
        this.refreshPromise = null;
      });
    }
    return this.refreshPromise;
  }

  async request(path, { method = "GET", body = null, query = null, skipUnauthorizedHandler = false, _isRetry = false } = {}) {
    let url = `${this.baseUrl}${path}`;

    if (query) {
      const params = new URLSearchParams();
      Object.entries(query).forEach(([k, v]) => {
        if (v !== undefined && v !== null && String(v).trim() !== "") {
          params.append(k, String(v).trim());
        }
      });
      const qs = params.toString();
      if (qs) url += `?${qs}`;
    }

    const headers = {};
    const token = this.tokenProvider();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
    if (body !== null) {
      headers["Content-Type"] = "application/json; charset=utf-8";
    }

    const response = await fetch(url, {
      method,
      headers,
      credentials: "include",
      body: body !== null ? JSON.stringify(body) : undefined,
    });

    const data = await response.json().catch(() => ({}));

    if (!response.ok) {
      if (response.status === 401 && !_isRetry && !this.#isAuthEndpoint(path)) {
        try {
          await this.#refreshAccessToken();
          return await this.request(path, {
            method,
            body,
            query,
            skipUnauthorizedHandler,
            _isRetry: true,
          });
        } catch (refreshErr) {
          if (!skipUnauthorizedHandler) {
            this.unauthorizedHandler(refreshErr.message || "Сессия истекла");
          }
          throw refreshErr;
        }
      }

      if (response.status === 401 && token && !skipUnauthorizedHandler && !this.#isAuthEndpoint(path)) {
        this.unauthorizedHandler(data.error || "Сессия истекла");
      }
      throw new ApiError(response.status, data.error || `Ошибка сервера (${response.status})`);
    }

    return data;
  }

  // Auth
  login(email, password) {
    return this.request("/auth/login", { method: "POST", body: { email, password } });
  }

  register(payload) {
    return this.request("/auth/register", { method: "POST", body: payload });
  }

  logout() {
    return this.request("/auth/logout", { method: "POST" });
  }

  me() {
    return this.request("/auth/me");
  }

  // Tasks
  listTasks(filters) {
    return this.request("/tasks", { query: filters });
  }

  createTask(payload) {
    return this.request("/tasks", { method: "POST", body: payload });
  }

  updateTask(taskId, payload) {
    return this.request(`/tasks/${taskId}`, { method: "PUT", body: payload });
  }

  assignMaster(taskId, masterProfileId) {
    return this.request(`/tasks/${taskId}/assign`, {
      method: "POST",
      body: { master_profile_id: Number(masterProfileId) },
    });
  }

  markTaskCompleted(taskId) {
    return this.request(`/tasks/${taskId}/mark-completed`, { method: "POST" });
  }

  completeTask(taskId) {
    return this.request(`/tasks/${taskId}/complete`, { method: "POST" });
  }

  cancelTask(taskId) {
    return this.request(`/tasks/${taskId}/cancel`, { method: "POST" });
  }

  // Bids
  createBid(taskId, payload) {
    return this.request(`/tasks/${taskId}/bids`, { method: "POST", body: payload });
  }

  listTaskBids(taskId) {
    return this.request(`/tasks/${taskId}/bids`);
  }

  // Masters & Services
  getMyMasterProfile() {
    return this.request("/master-profile/me");
  }

  upsertMyMasterProfile(payload) {
    return this.request("/master-profile/me", { method: "PUT", body: payload });
  }

  getPublicMasterProfile(id) {
    return this.request(`/masters/${id}`);
  }

  listMasterServices(filters) {
    return this.request("/master-services", { query: filters });
  }

  createMasterService(payload) {
    return this.request("/master-services", { method: "POST", body: payload });
  }

  updateMasterService(id, payload) {
    return this.request(`/master-services/${id}`, { method: "PUT", body: payload });
  }

  deleteMasterService(id) {
    return this.request(`/master-services/${id}`, { method: "DELETE" });
  }

  // Reviews
  createReview(taskId, payload) {
    return this.request(`/tasks/${taskId}/review`, { method: "POST", body: payload });
  }

  updateReview(reviewId, payload) {
    return this.request(`/reviews/${reviewId}`, { method: "PUT", body: payload });
  }

  deleteReview(reviewId) {
    return this.request(`/reviews/${reviewId}`, { method: "DELETE" });
  }
}

export const api = new HttpClient("/api/v1");
