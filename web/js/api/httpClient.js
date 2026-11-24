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
  }

  setTokenProvider(fn) {
    this.tokenProvider = fn;
  }

  async request(path, { method = "GET", body = null, query = null } = {}) {
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
      body: body !== null ? JSON.stringify(body) : undefined,
    });

    const data = await response.json().catch(() => ({}));

    if (!response.ok) {
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

  // Tasks
  listTasks(filters) {
    return this.request("/tasks", { query: filters });
  }

  createTask(payload) {
    return this.request("/tasks", { method: "POST", body: payload });
  }

  assignMaster(taskId, masterId) {
    return this.request(`/tasks/${taskId}/assign`, { method: "POST", body: { master_id: Number(masterId) } });
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

  // Masters
  listMasters(filters) {
    return this.request("/masters", { query: filters });
  }

  createMasterProfile(payload) {
    return this.request("/masters", { method: "POST", body: payload });
  }

  deleteMasterProfile(id) {
    return this.request(`/masters/${id}`, { method: "DELETE" });
  }
}

export const api = new HttpClient("/api/v1");
