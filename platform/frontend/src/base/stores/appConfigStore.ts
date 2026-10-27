import { create } from "zustand";

export interface AppConfigState {
  appName: string;
  apiBaseUrl: string;
  pageSize: number;
  setPageSize: (pageSize: number) => void;
}

export const useAppConfigStore = create<AppConfigState>((set) => ({
  appName: "WFAgent Base",
  apiBaseUrl: (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "http://localhost:8080/v1",
  pageSize: 10,
  setPageSize: (pageSize) => set({ pageSize }),
}));
