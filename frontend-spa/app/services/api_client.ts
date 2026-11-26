const DEFAULT_API_BASE_URL = 'http://localhost:8080/api/v1';

let inMemoryCSRFToken = '';

export function getCSRFToken(): string {
  if (!inMemoryCSRFToken && typeof window !== 'undefined') {
    inMemoryCSRFToken = sessionStorage.getItem('csrf_token') || '';
  }
  return inMemoryCSRFToken;
}

export function setCSRFToken(token: string): void {
  inMemoryCSRFToken = token || '';
  if (typeof window !== 'undefined') {
    if (token) {
      sessionStorage.setItem('csrf_token', token);
    } else {
      sessionStorage.removeItem('csrf_token');
    }
  }
}

export class ApiError extends Error {
  status: number;
  data?: any;

  constructor(status: number, message: string, data?: any) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;
  }
}

export interface ApiResponse<T = any> {
  success: boolean;
  message?: string;
  data?: T;
  error?: string;
}

export function getApiBaseUrl(): string {
  const envUrl = import.meta.env.VITE_API_URL;
  if (envUrl && typeof envUrl === 'string') {
    return envUrl.replace(/\/+$/, '');
  }
  return DEFAULT_API_BASE_URL;
}

export async function request<T = any>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const baseUrl = getApiBaseUrl();
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  const url = `${baseUrl}${normalizedPath}`;

  const headers = new Headers(options.headers || {});

  const method = (options.method || 'GET').toUpperCase();
  const isMutatingMethod = ['POST', 'PUT', 'PATCH', 'DELETE'].includes(method);

  const csrfToken = getCSRFToken();
  if (isMutatingMethod && csrfToken) {
    headers.set('X-CSRF-Token', csrfToken);
  }

  if (options.body && !(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(url, {
    ...options,
    method,
    headers,
    credentials: 'include',
  });

  if (response.status === 204) {
    return undefined as unknown as T;
  }

  let jsonResult: ApiResponse<T>;
  try {
    jsonResult = await response.json();
  } catch {
    if (!response.ok) {
      throw new ApiError(response.status, `HTTP ${response.status}: ${response.statusText}`);
    }
    return undefined as unknown as T;
  }

  if (jsonResult && typeof jsonResult === 'object' && 'data' in jsonResult) {
    const responseData = jsonResult.data as any;
    if (responseData && typeof responseData === 'object') {
      const token = responseData.csrfToken || responseData.csrf_token;
      if (typeof token === 'string' && token) {
        setCSRFToken(token);
      }
    }
  }

  if (!response.ok || jsonResult.success === false) {
    const errorMessage = jsonResult.error || jsonResult.message || `Request failed with status ${response.status}`;
    throw new ApiError(response.status, errorMessage, jsonResult.data);
  }

  return (jsonResult.data !== undefined ? jsonResult.data : jsonResult) as T;
}

export const api = {
  get<T = any>(path: string, options?: RequestInit): Promise<T> {
    return request<T>(path, { ...options, method: 'GET' });
  },
  post<T = any>(path: string, body?: any, options?: RequestInit): Promise<T> {
    return request<T>(path, {
      ...options,
      method: 'POST',
      body: body !== undefined ? (body instanceof FormData ? body : JSON.stringify(body)) : undefined,
    });
  },
  put<T = any>(path: string, body?: any, options?: RequestInit): Promise<T> {
    return request<T>(path, {
      ...options,
      method: 'PUT',
      body: body !== undefined ? (body instanceof FormData ? body : JSON.stringify(body)) : undefined,
    });
  },
  patch<T = any>(path: string, body?: any, options?: RequestInit): Promise<T> {
    return request<T>(path, {
      ...options,
      method: 'PATCH',
      body: body !== undefined ? (body instanceof FormData ? body : JSON.stringify(body)) : undefined,
    });
  },
  delete<T = any>(path: string, options?: RequestInit): Promise<T> {
    return request<T>(path, { ...options, method: 'DELETE' });
  },
};
