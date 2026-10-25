import type { LoginResponse } from "@/lib/types";
import { request } from "@/lib/api/client";

const CURRENT_USER_STORAGE_KEY = "wbd_current_user";

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
  } catch {
    // Ignore storage errors
  }
}

export const authApi = {
  login: (username: string, password: string) =>
    request<{ data: LoginResponse }>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }).then((body) => {
      if (typeof window !== "undefined" && body.data) {
        try {
          window.localStorage.setItem(
            CURRENT_USER_STORAGE_KEY,
            JSON.stringify(body.data),
          );
        } catch {
          // Ignore storage errors
        }
      }
      return body.data;
    }),

  logout: () =>
    request<void>("/api/v1/auth/logout", { method: "POST" }).finally(() => {
      clearStoredCurrentUser();
    }),
};
