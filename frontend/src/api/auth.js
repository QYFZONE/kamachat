import { postJSON } from "./http";

// Public routes defined in setupPublicRoutes().
export function loginByPassword(payload) {
  return postJSON("/auth/login", payload);
}

export function loginBySms(payload) {
  return postJSON("/auth/sms/login", payload);
}

export function registerAccount(payload) {
  return postJSON("/auth/register", payload);
}

export function sendSmsCode(payload) {
  return postJSON("/sms/send", payload);
}
