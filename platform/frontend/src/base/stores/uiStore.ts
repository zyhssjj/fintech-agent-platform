import { create } from "zustand";

export type ThemeMode = "light" | "dark";
export type NoticeType = "success" | "error" | "warning" | "info";

export interface GlobalNotice {
  id: string;
  type: NoticeType;
  title: string;
  message?: string;
}

export interface UIState {
  sidebarCollapsed: boolean;
  theme: ThemeMode;
  loadingCount: number;
  notices: GlobalNotice[];
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setTheme: (theme: ThemeMode) => void;
  beginLoading: () => void;
  endLoading: () => void;
  notify: (notice: Omit<GlobalNotice, "id">) => void;
  removeNotice: (id: string) => void;
}

const THEME_KEY = "wfagent.theme";
const canUseStorage = () => typeof window !== "undefined" && Boolean(window.localStorage);
const initialTheme = canUseStorage() ? (window.localStorage.getItem(THEME_KEY) as ThemeMode | null) ?? "light" : "light";

export const useUIStore = create<UIState>((set) => ({
  sidebarCollapsed: false,
  theme: initialTheme,
  loadingCount: 0,
  notices: [],
  toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
  setSidebarCollapsed: (sidebarCollapsed) => set({ sidebarCollapsed }),
  setTheme: (theme) => {
    if (canUseStorage()) {
      window.localStorage.setItem(THEME_KEY, theme);
    }
    document.documentElement.dataset.theme = theme;
    set({ theme });
  },
  beginLoading: () => set((state) => ({ loadingCount: state.loadingCount + 1 })),
  endLoading: () => set((state) => ({ loadingCount: Math.max(0, state.loadingCount - 1) })),
  notify: (notice) => {
    const id = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
    set((state) => ({ notices: [...state.notices, { ...notice, id }].slice(-5) }));
    window.setTimeout(() => {
      useUIStore.getState().removeNotice(id);
    }, 4500);
  },
  removeNotice: (id) => set((state) => ({ notices: state.notices.filter((notice) => notice.id !== id) })),
}));
