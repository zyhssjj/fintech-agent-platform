import { create } from "zustand";

export interface OpenTab {
  path: string;
  title: string;
}

interface NavigationState {
  selectedModuleId: string;
  filters: Record<string, unknown>;
  tabs: OpenTab[];
  setSelectedModuleId: (selectedModuleId: string) => void;
  openTab: (tab: OpenTab) => void;
  closeTab: (path: string) => void;
  setFilter: (key: string, value: unknown) => void;
  clearFilters: () => void;
}

export const useNavigationStore = create<NavigationState>((set) => ({
  selectedModuleId: "dashboard",
  filters: {},
  tabs: [{ path: "/", title: "工作台总览" }],
  setSelectedModuleId: (selectedModuleId) => set({ selectedModuleId }),
  openTab: (tab) =>
    set((state) => ({
      selectedModuleId: tab.path,
      tabs: state.tabs.some((item) => item.path === tab.path) ? state.tabs : [...state.tabs, tab],
    })),
  closeTab: (path) =>
    set((state) => {
      const nextTabs = state.tabs.filter((tab) => tab.path !== path);
      return { tabs: nextTabs.length ? nextTabs : [{ path: "/", title: "工作台总览" }] };
    }),
  setFilter: (key, value) => set((state) => ({ filters: { ...state.filters, [key]: value } })),
  clearFilters: () => set({ filters: {} }),
}));
