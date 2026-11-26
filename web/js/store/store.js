const STORAGE_KEY = "bboard_auth_session";

class Store {
  constructor() {
    const saved = this.#loadSession();
    this.state = {
      token: saved?.token || null,
      user: saved?.user || null,
      activeTab: "tasks", // "tasks" | "masters"
      filters: {
        category: "",
        city: "",
        status: "open",
      },
      tasks: [],
      masters: [],
      myProfile: null,
      myServices: [],
      loading: false,
    };
    this.listeners = new Set();
  }

  subscribe(listener) {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  setState(patch) {
    this.state = { ...this.state, ...patch };
    this.listeners.forEach((fn) => fn(this.state));
  }

  setSession(token, user) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ token, user }));
    this.setState({ token, user });
  }

  clearSession() {
    localStorage.removeItem(STORAGE_KEY);
    this.setState({
      token: null,
      user: null,
      myProfile: null,
      myServices: [],
      activeTab: this.state.activeTab === "cabinet" ? "tasks" : this.state.activeTab,
    });
  }

  #loadSession() {
    try {
      return JSON.parse(localStorage.getItem(STORAGE_KEY));
    } catch {
      return null;
    }
  }
}

export const store = new Store();