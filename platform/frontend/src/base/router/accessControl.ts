import type { UserInfo } from "@/base/api";

export function hasAnyRole(user: UserInfo | null, roles?: string[]) {
  if (!roles?.length) return true;
  if (!user) return false;
  if (user.roles.includes("SYS_ADMIN") || user.roles.includes("*")) return true;
  return roles.some((role) => user.roles.includes(role));
}

export function hasAnyPermission(user: UserInfo | null, permissions?: string[]) {
  if (!permissions?.length) return true;
  if (!user) return false;
  if (user.permissions.includes("*")) return true;
  return permissions.some((permission) => user.permissions.includes(permission));
}
