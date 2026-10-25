import type { LoginResponse } from "@/lib/types";
import { request } from "@/lib/api/client";

let inflightMe: Promise<LoginResponse> | null = null;

export const authApi = {
  login: async (username: string, password: string) => {
    const data = await request<{ data: LoginResponse }>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }).then((body) => body.data);

    if (typeof window !== "undefined") {
      localStorage.setItem("wbd_user", JSON.stringify(data));
      window.dispatchEvent(new Event("auth-changed"));
    }
    return data;
  },

  logout: async () => {
    try {
      await request<void>("/api/v1/auth/logout", { method: "POST" });
    } finally {
      if (typeof window !== "undefined") {
        localStorage.removeItem("wbd_user");
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
        const data = await request<{ data: LoginResponse }>("/api/v1/auth/me").then(
          (body) => body.data,
        );
        if (typeof window !== "undefined") {
          localStorage.setItem("wbd_user", JSON.stringify(data));
        }
        return data;
      } catch (err) {
        if (typeof window !== "undefined") {
          localStorage.removeItem("wbd_user");
        }
        throw err;
      } finally {
        inflightMe = null;
      }
    })();

    return inflightMe;
  },
};
