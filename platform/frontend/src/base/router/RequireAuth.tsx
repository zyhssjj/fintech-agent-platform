import React from "react";
import { Navigate, useLocation } from "react-router-dom";
import { hasAnyPermission, hasAnyRole } from "@/base/router/accessControl";
import { useAuthStore } from "@/base/stores/authStore";
import { ForbiddenView } from "@/base/views/ForbiddenView";

interface RequireAuthProps {
  children: React.ReactNode;
  requiredRoles?: string[];
  requiredPermissions?: string[];
}

export function RequireAuth({ children, requiredRoles, requiredPermissions }: RequireAuthProps) {
  const location = useLocation();
  const user = useAuthStore((state) => state.user);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }

  if (!hasAnyRole(user, requiredRoles) || !hasAnyPermission(user, requiredPermissions)) {
    return <ForbiddenView />;
  }

  return <>{children}</>;
}
