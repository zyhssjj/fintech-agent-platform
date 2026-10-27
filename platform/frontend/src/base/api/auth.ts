import type {
  LoginRequest,
  ForgotPasswordRequest,
  ForgotPasswordResponse,
  PasswordChangeRequest,
  RegisterRequest,
  RegisterResponse,
  ResetPasswordRequest,
  TokenResponse,
  UserInfo,
} from "@/base/api/contracts";
import { request } from "@/base/api/request";
import { authStore } from "@/base/stores/authStore";

// 本文件封装认证授权接口 AUTH-001~AUTH-006。
export const authApi = {
  async login(payload: LoginRequest) {
    const token = await request.post<TokenResponse>("/auth/login", payload, { auth: false });
    authStore.setSession(token);
    return token;
  },
  refresh(refreshToken: string) {
    return request.post<TokenResponse>("/auth/refresh", { refresh_token: refreshToken }, { auth: false });
  },
  logout() {
    return request.post<{ status: string }>("/auth/logout").finally(() => authStore.clearSession());
  },
  me() {
    return request.get<UserInfo>("/auth/me");
  },
  changePassword(payload: PasswordChangeRequest) {
    return request.put<{ status: string }>("/auth/password", payload);
  },
  forgotPassword(payload: ForgotPasswordRequest) {
    return request.post<ForgotPasswordResponse>("/auth/password/forgot", payload, { auth: false });
  },
  resetPassword(payload: ResetPasswordRequest) {
    return request.post<{ status: string }>("/auth/password/reset", payload, { auth: false });
  },
  register(payload: RegisterRequest) {
    return request.post<RegisterResponse>("/auth/register", payload, { auth: false });
  },
};
