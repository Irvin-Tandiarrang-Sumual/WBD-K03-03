import type { LoginResponse } from "@/lib/types";
import { request } from "@/lib/api/client";

const CURRENT_USER_STORAGE_KEY = "wbd_user";

export function getStoredCurrentUser(): LoginResponse | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.localStorage.getItem(CURRENT_USER_STORAGE_KEY);
    return raw ? (JSON.parse(raw) as LoginResponse) : null;
  } catch {
    return null;
  }
}

export function clearStoredCurrentUser(): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.removeItem(CURRENT_USER_STORAGE_KEY);
    window.dispatchEvent(new Event("auth-changed"));
  } catch {
    // Ignore storage errors
  }
}

let inflightMe: Promise<LoginResponse> | null = null;

export const authApi = {
  login: async (username: string, password: string) => {
    const data = await request<{ data: LoginResponse }>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }).then((body) => body.data);

    if (typeof window !== "undefined") {
      localStorage.setItem(CURRENT_USER_STORAGE_KEY, JSON.stringify(data));
      window.dispatchEvent(new Event("auth-changed"));
    }
    return data;
  },

  logout: async () => {
    try {
      await request<void>("/api/v1/auth/logout", { method: "POST" });
    } finally {
      if (typeof window !== "undefined") {
        localStorage.removeItem(CURRENT_USER_STORAGE_KEY);
        window.dispatchEvent(new Event("auth-changed"));
      }
    }
  },

  me: async () => {
    if (inflightMe) {
      return inflightMe;
    }

    inflightMe = (async () => {
      try {
        const data = await request<{ data: LoginResponse }>(
          "/api/v1/auth/me",
        ).then((body) => body.data);
        if (typeof window !== "undefined") {
          localStorage.setItem(CURRENT_USER_STORAGE_KEY, JSON.stringify(data));
        }
        return data;
      } catch (err) {
        if (typeof window !== "undefined") {
          localStorage.removeItem(CURRENT_USER_STORAGE_KEY);
        }
        throw err;
      } finally {
        inflightMe = null;
      }
    })();

    return inflightMe;
  },
};
