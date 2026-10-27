import { create } from "zustand";

interface PermissionState {
  roles: string[];
  permissions: string[];
  setAccess: (roles: string[], permissions: string[]) => void;
  clearAccess: () => void;
}

export const usePermissionStore = create<PermissionState>((set) => ({
  roles: [],
  permissions: [],
  setAccess: (roles, permissions) => set({ roles, permissions }),
  clearAccess: () => set({ roles: [], permissions: [] }),
}));
