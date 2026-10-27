import type { ApiEnvelope, TokenResponse } from "@/base/api/contracts";
import { authStore } from "@/base/stores/authStore";
import { useUIStore } from "@/base/stores/uiStore";

const normalizeBaseURL = (value: string | undefined) => {
  const baseURL = value?.trim() || "/v1";
  return baseURL === "/" ? "" : baseURL.replace(/\/+$/, "");
};

const API_BASE_URL = normalizeBaseURL(import.meta.env.VITE_API_BASE_URL as string | undefined);

export class ApiError extends Error {
  code: number;
  status: number;
  requestId?: string;

  constructor(message: string, code: number, status: number, requestId?: string) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
    this.requestId = requestId;
  }
}

export interface RequestOptions extends Omit<RequestInit, "body"> {
  auth?: boolean;
  query?: object;
  body?: unknown;
  retryOnAuth?: boolean;
  silent?: boolean;
}

const buildURL = (path: string, query?: RequestOptions["query"]) => {
  const rawURL = path.startsWith("http") ? path : `${API_BASE_URL}${path}`;
  // URL needs an absolute base, while fetch deliberately receives a relative
  // same-origin URL when the Vite/Express proxy is in use.
  const url = new URL(rawURL, window.location.origin);
  Object.entries(query ?? {}).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") return;
    if (Array.isArray(value)) {
      value.forEach((item) => url.searchParams.append(key, String(item)));
      return;
    }
    url.searchParams.set(key, String(value));
  });
  return rawURL.startsWith("http")
    ? url.toString()
    : `${url.pathname}${url.search}${url.hash}`;
};

async function refreshAccessToken(): Promise<boolean> {
  const { refreshToken } = authStore.getSnapshot();
  if (!refreshToken) return false;

  const resp = await fetch(buildURL("/auth/refresh"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });

  if (!resp.ok) {
    authStore.clearSession();
    return false;
  }

  const envelope = await resp.json() as ApiEnvelope<TokenResponse>;
  if (!envelope.data?.access_token) {
    authStore.clearSession();
    return false;
  }
  authStore.setAccessToken(envelope.data.access_token);
  return true;
}

async function execute<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { auth = true, query, body, retryOnAuth = true, silent = false, headers, ...rest } = options;
  const requestHeaders = new Headers(headers);
  const isFormData = typeof FormData !== "undefined" && body instanceof FormData;

  if (!silent) {
    useUIStore.getState().beginLoading();
  }

  try {
    if (body !== undefined && !isFormData && !requestHeaders.has("Content-Type")) {
      requestHeaders.set("Content-Type", "application/json");
    }
    if (auth) {
      const { accessToken } = authStore.getSnapshot();
      if (accessToken) {
        requestHeaders.set("Authorization", `Bearer ${accessToken}`);
      }
    }

    const resp = await fetch(buildURL(path, query), {
      ...rest,
      headers: requestHeaders,
      body: body === undefined ? undefined : isFormData ? body : JSON.stringify(body),
    });

    if (resp.status === 401 && auth && retryOnAuth) {
      const refreshed = await refreshAccessToken();
      if (refreshed) {
        return execute<T>(path, { ...options, retryOnAuth: false });
      }
    }

    if (resp.status === 204) {
      return undefined as T;
    }

    const envelope = await resp.json().catch(() => ({ code: resp.status, message: resp.statusText })) as ApiEnvelope<T>;
    if (!resp.ok || envelope.code >= 10001) {
      throw new ApiError(envelope.message || resp.statusText, envelope.code, resp.status, envelope.request_id);
    }
    return envelope.data as T;
  } catch (error) {
    if (!silent) {
      useUIStore.getState().notify({
        type: "error",
        title: "请求失败",
        message: error instanceof Error ? error.message : "后端接口暂时不可用",
      });
    }
    throw error;
  } finally {
    if (!silent) {
      useUIStore.getState().endLoading();
    }
  }
}

export const request = {
  get: <T>(path: string, query?: RequestOptions["query"], options?: RequestOptions) =>
    execute<T>(path, { ...options, method: "GET", query }),
  post: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    execute<T>(path, { ...options, method: "POST", body }),
  put: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    execute<T>(path, { ...options, method: "PUT", body }),
  patch: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    execute<T>(path, { ...options, method: "PATCH", body }),
  delete: <T = void>(path: string, options?: RequestOptions) =>
    execute<T>(path, { ...options, method: "DELETE" }),
  upload: <T>(path: string, formData: FormData, options?: RequestOptions) =>
    execute<T>(path, { ...options, method: "POST", body: formData }),
};
