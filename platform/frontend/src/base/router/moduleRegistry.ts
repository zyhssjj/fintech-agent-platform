export type ModuleCategory = "overview" | "fundraising" | "investing" | "risk_control" | "system";

export interface ModuleRoute {
  id: string;
  name: string;
  category: ModuleCategory;
  path: string;
  iconName: string;
  requiredRoles?: string[];
  requiredPermissions?: string[];
}

export const categoryLabels: Record<ModuleCategory, string> = {
  overview: "工作总览",
  fundraising: "募资与 LP 协同",
  investing: "投资与项目管理",
  risk_control: "风控与智能分析",
  system: "系统管理",
};

const internalRoles = ["FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "OPS_MANAGER"];
const reviewerRoles = [...internalRoles, "REVIEWER"];
const lpVisibleRoles = [...reviewerRoles, "LP"];

export const moduleRoutes: ModuleRoute[] = [
  { id: "meetings", name: "会议管理", category: "overview", path: "meetings", iconName: "Calendar", requiredRoles: internalRoles },
  { id: "communication", name: "通讯中心", category: "overview", path: "agents/communication", iconName: "MessageSquare", requiredRoles: [...internalRoles, "LP"] },
  { id: "knowledge", name: "知识合规库", category: "overview", path: "agents/knowledge", iconName: "ShieldAlert", requiredRoles: reviewerRoles },

  { id: "approvals", name: "内部协同审批", category: "fundraising", path: "approvals", iconName: "FileCheck", requiredRoles: reviewerRoles },
  { id: "new_fund", name: "新设基金募资", category: "fundraising", path: "fundraising/funds/new", iconName: "TrendingUp", requiredRoles: ["FUND_MANAGER", "INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER"] },
  { id: "fund_ops", name: "存量基金运营", category: "fundraising", path: "fundraising/funds", iconName: "Layers", requiredRoles: lpVisibleRoles },
  { id: "lp_manage", name: "LP 管理", category: "fundraising", path: "fundraising/lps", iconName: "Users", requiredRoles: ["FUND_MANAGER", "FINANCE_MANAGER", "LP"] },

  { id: "pre_invest", name: "投前开发", category: "investing", path: "investment/projects", iconName: "Compass", requiredRoles: ["FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "REVIEWER"] },
  { id: "due_diligence", name: "投中尽调决策", category: "investing", path: "investment/due-diligence", iconName: "FileSearch", requiredRoles: ["FUND_MANAGER", "INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "REVIEWER"] },
  { id: "post_invest", name: "投后监控", category: "investing", path: "projects/post-investment", iconName: "Activity", requiredRoles: lpVisibleRoles },
  { id: "exit_manage", name: "退出管理", category: "investing", path: "exit", iconName: "LogOut", requiredRoles: lpVisibleRoles },

  { id: "risk_val", name: "风控估值", category: "risk_control", path: "investment/valuation", iconName: "Percent", requiredRoles: ["FUND_MANAGER", "INVEST_MANAGER", "POST_INVEST_MANAGER", "RISK_MANAGER", "LEGAL_MANAGER", "FINANCE_MANAGER", "REVIEWER"] },
  { id: "ai_report", name: "AI 智能体报告", category: "risk_control", path: "agents/reports", iconName: "Cpu", requiredRoles: reviewerRoles },
  { id: "disclosure", name: "报表披露", category: "risk_control", path: "exit/lp-disclosures", iconName: "FileText", requiredRoles: lpVisibleRoles },

  { id: "audit_logs", name: "操作日志", category: "system", path: "system/audit-logs", iconName: "ClipboardList", requiredRoles: lpVisibleRoles },
  { id: "notifications", name: "通知中心", category: "system", path: "system/notifications", iconName: "Bell", requiredRoles: lpVisibleRoles },
  { id: "config", name: "API 配置", category: "system", path: "system/config", iconName: "Sliders", requiredRoles: ["SYS_ADMIN"] },
];

export const dashboardRoute = {
  id: "dashboard",
  name: "工作台总览",
  category: "overview" as const,
  path: "/",
  iconName: "LayoutDashboard",
};

export function getModuleByPath(pathname: string) {
  const normalized = pathname.replace(/^\/+/, "");
  if (pathname === "/" || normalized === "dashboard") {
    return dashboardRoute;
  }
  if (normalized === "profile") {
    return { ...dashboardRoute, id: "profile", name: "个人中心", path: "profile" };
  }
  return moduleRoutes.find((route) => normalized === route.path || normalized.startsWith(`${route.path}/`));
}
