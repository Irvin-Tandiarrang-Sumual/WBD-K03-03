const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    cache: options.cache ?? "no-store",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
  });

  const body: unknown = await response.json().catch(() => ({}));
  const errorBody = body as {
    message?: string;
    error?: string;
    success?: boolean;
  };

  if (!response.ok || errorBody.success === false) {
    throw new Error(
      errorBody.message ||
        errorBody.error ||
        `Request failed with status ${response.status}`,
    );
  }

  return body as T;
}

export function unwrapPaged<T>(body: { data?: T[] }): T[] {
  return body.data || [];
}
