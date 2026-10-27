import type { RouteObject } from "react-router-dom";
import type React from "react";
import EmptyModuleView from "@/components/EmptyModuleView";
import DashboardView from "@/components/DashboardView";
import { moduleRoutes } from "@/base/router/moduleRegistry";
import { RequireAuth } from "@/base/router/RequireAuth";
import { AuditLogView } from "@/base/views/AuditLogView";
import { ForbiddenView } from "@/base/views/ForbiddenView";
import { ForgotPasswordView } from "@/base/views/ForgotPasswordView";
import { LoginView } from "@/base/views/LoginView";
import { NotFoundView } from "@/base/views/NotFoundView";
import { NotificationCenterView } from "@/base/views/NotificationCenterView";
import { ProfileView } from "@/base/views/ProfileView";
import { RegisterView } from "@/base/views/RegisterView";
import { ServerErrorView } from "@/base/views/ServerErrorView";
import { SystemConfigView } from "@/base/views/SystemConfigView";
import { ApprovalsView } from "@/module-agent/views/ApprovalsView";
import { CommunicationCenterView } from "@/module-agent/views/CommunicationCenterView";
import { KnowledgeView } from "@/module-agent/views/KnowledgeView";

const systemElements: Record<string, React.ReactElement> = {
  audit_logs: <AuditLogView />,
  notifications: <NotificationCenterView />,
  config: <SystemConfigView />,
  approvals: <ApprovalsView />,
  communication: <CommunicationCenterView />,
  knowledge: <KnowledgeView />,
};

export const moduleRouteObjects: RouteObject[] = moduleRoutes.map((module) => ({
  path: module.path,
  element: (
    <RequireAuth requiredRoles={module.requiredRoles} requiredPermissions={module.requiredPermissions}>
      {systemElements[module.id] ?? <EmptyModuleView title={module.name} />}
    </RequireAuth>
  ),
}));

export const appRoutes = {
  dashboard: (
    <RequireAuth>
      <DashboardView />
    </RequireAuth>
  ),
  modules: moduleRouteObjects,
  profile: (
    <RequireAuth>
      <ProfileView />
    </RequireAuth>
  ),
  login: <LoginView />,
  register: <RegisterView />,
  forgotPassword: <ForgotPasswordView />,
  forbidden: <ForbiddenView />,
  serverError: <ServerErrorView />,
  notFound: <NotFoundView />,
};
