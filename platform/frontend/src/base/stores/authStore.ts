import { create } from "zustand";
import type { TokenResponse, UserInfo } from "@/base/api/contracts";
import { usePermissionStore } from "@/base/stores/permissionStore";

const ACCESS_TOKEN_KEY = "wfagent.access_token";
const REFRESH_TOKEN_KEY = "wfagent.refresh_token";
const USER_INFO_KEY = "wfagent.user_info";

export interface AuthSnapshot {
  accessToken: string;
  refreshToken: string;
  user: UserInfo | null;
}

interface AuthState extends AuthSnapshot {
  isAuthenticated: boolean;
  setSession: (token: TokenResponse) => void;
  setAccessToken: (accessToken: string) => void;
  setUser: (user: UserInfo) => void;
  clearSession: () => void;
}

const canUseStorage = () => typeof window !== "undefined" && Boolean(window.localStorage);

function readInitialState(): AuthSnapshot {
  if (!canUseStorage()) {
    return { accessToken: "", refreshToken: "", user: null };
  }

  const rawUser = window.localStorage.getItem(USER_INFO_KEY);
  return {
    accessToken: window.localStorage.getItem(ACCESS_TOKEN_KEY) ?? "",
    refreshToken: window.localStorage.getItem(REFRESH_TOKEN_KEY) ?? "",
    user: rawUser ? JSON.parse(rawUser) as UserInfo : null,
  };
}

function persistSession(snapshot: AuthSnapshot) {
  if (!canUseStorage()) return;
  window.localStorage.setItem(ACCESS_TOKEN_KEY, snapshot.accessToken);
  if (snapshot.refreshToken) {
    window.localStorage.setItem(REFRESH_TOKEN_KEY, snapshot.refreshToken);
  }
  if (snapshot.user) {
    window.localStorage.setItem(USER_INFO_KEY, JSON.stringify(snapshot.user));
  }
}

function clearPersistedSession() {
  if (!canUseStorage()) return;
  window.localStorage.removeItem(ACCESS_TOKEN_KEY);
  window.localStorage.removeItem(REFRESH_TOKEN_KEY);
  window.localStorage.removeItem(USER_INFO_KEY);
}

const initialState = readInitialState();

export const useAuthStore = create<AuthState>((set) => ({
  ...initialState,
  isAuthenticated: Boolean(initialState.accessToken),

  setSession: (token) => {
    const next = {
      accessToken: token.access_token,
      refreshToken: token.refresh_token ?? useAuthStore.getState().refreshToken,
      user: token.user_info ?? useAuthStore.getState().user,
    };
    persistSession(next);
    usePermissionStore.getState().setAccess(next.user?.roles ?? [], next.user?.permissions ?? []);
    set({ ...next, isAuthenticated: Boolean(next.accessToken) });
  },

  setAccessToken: (accessToken) => {
    const current = useAuthStore.getState();
    const next = { accessToken, refreshToken: current.refreshToken, user: current.user };
    persistSession(next);
    set({ accessToken, isAuthenticated: Boolean(accessToken) });
  },

  setUser: (user) => {
    if (canUseStorage()) {
      window.localStorage.setItem(USER_INFO_KEY, JSON.stringify(user));
    }
    usePermissionStore.getState().setAccess(user.roles ?? [], user.permissions ?? []);
    set({ user });
  },

  clearSession: () => {
    clearPersistedSession();
    usePermissionStore.getState().clearAccess();
    set({ accessToken: "", refreshToken: "", user: null, isAuthenticated: false });
  },
}));

if (initialState.user) {
  usePermissionStore.getState().setAccess(initialState.user.roles ?? [], initialState.user.permissions ?? []);
}

export const authStore = {
  getSnapshot(): AuthSnapshot {
    const { accessToken, refreshToken, user } = useAuthStore.getState();
    return { accessToken, refreshToken, user };
  },
  setSession(token: TokenResponse) {
    useAuthStore.getState().setSession(token);
  },
  setAccessToken(accessToken: string) {
    useAuthStore.getState().setAccessToken(accessToken);
  },
  setUser(user: UserInfo) {
    useAuthStore.getState().setUser(user);
  },
  clearSession() {
    useAuthStore.getState().clearSession();
  },
};
