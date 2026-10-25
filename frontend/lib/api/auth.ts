import type { LoginResponse } from "@/lib/types";
import { request } from "@/lib/api/client";

export const authApi = {
  login: (username: string, password: string) =>
    request<{ data: LoginResponse }>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }).then((body) => body.data),

  logout: () => request<void>("/api/v1/auth/logout", { method: "POST" }),
};
