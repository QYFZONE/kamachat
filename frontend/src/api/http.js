const API_BASE_URL = import.meta.env.VITE_API_BASE_URL?.replace(/\/+$/, "") || "";
const WS_BASE_URL = import.meta.env.VITE_WS_BASE_URL?.replace(/\/+$/, "") || "";

class ApiError extends Error {
  constructor(message, payload) {
    super(message);
    this.name = "ApiError";
    this.payload = payload;
  }
}

export function buildUrl(path) {
  if (/^https?:\/\//.test(path)) {
    return path;
  }

  if (!API_BASE_URL) {
    return path;
  }

  return `${API_BASE_URL}${path}`;
}

export function buildWebSocketUrl(path) {
  if (/^wss?:\/\//.test(path)) {
    return path;
  }

  if (WS_BASE_URL) {
    return `${WS_BASE_URL}${path}`;
  }

  if (API_BASE_URL) {
    const apiUrl = new URL(API_BASE_URL);
    apiUrl.protocol = apiUrl.protocol === "https:" ? "wss:" : "ws:";
    apiUrl.pathname = path;
    apiUrl.search = "";
    return apiUrl.toString();
  }

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}${path}`;
}

export function resolveAssetUrl(path) {
  if (!path || /^https?:\/\//.test(path) || /^data:/.test(path)) {
    return path || "";
  }

  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  return API_BASE_URL ? `${API_BASE_URL}${normalizedPath}` : normalizedPath;
}

// 后端统一返回 JsonBack 结构：
// { code, message, data }。
// 业务失败也可能是 HTTP 200，
// 所以前端要以 body.code 为准。
async function parseJsonBack(response) {
  const rawText = await response.text();
  let result;

  try {
    result = JSON.parse(rawText);
  } catch (error) {
    throw new ApiError("服务返回异常，请稍后重试", {
      code: 0,
      message: rawText,
    });
  }

  if (!response.ok) {
    throw new ApiError(result.message || "请求失败，请稍后重试", result);
  }

  if (result.code !== 200) {
    throw new ApiError(result.message || "业务处理失败，请稍后重试", result);
  }

  return result;
}

export async function postJSON(path, payload) {
  const response = await fetch(buildUrl(path), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  return parseJsonBack(response);
}

export async function postFormData(path, formData) {
  const response = await fetch(buildUrl(path), {
    method: "POST",
    body: formData,
  });

  return parseJsonBack(response);
}

export { ApiError };
