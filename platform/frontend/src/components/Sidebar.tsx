import {
  Activity,
  Bell,
  Calendar,
  ChevronLeft,
  ChevronRight,
  ClipboardList,
  Compass,
  Cpu,
  FileCheck,
  FileSearch,
  FileText,
  Layers,
  LayoutDashboard,
  LogOut,
  MessageSquare,
  Percent,
  ShieldAlert,
  Sliders,
  Sparkles,
  TrendingUp,
  Users,
} from "lucide-react";
import { NavLink } from "react-router-dom";
import { categoryLabels, dashboardRoute, moduleRoutes, type ModuleCategory } from "@/base/router/moduleRegistry";
import { hasAnyPermission, hasAnyRole } from "@/base/router/accessControl";
import { useAuthStore } from "@/base/stores/authStore";

interface SidebarProps {
  collapsed: boolean;
  setCollapsed: (collapsed: boolean) => void;
}

const categories: ModuleCategory[] = ["overview", "fundraising", "investing", "risk_control", "system"];

function Icon({ name, className }: { name: string; className: string }) {
  const props = { className };
  switch (name) {
    case "Activity": return <Activity {...props} />;
    case "Bell": return <Bell {...props} />;
    case "Calendar": return <Calendar {...props} />;
    case "ClipboardList": return <ClipboardList {...props} />;
    case "Compass": return <Compass {...props} />;
    case "Cpu": return <Cpu {...props} />;
    case "FileCheck": return <FileCheck {...props} />;
    case "FileSearch": return <FileSearch {...props} />;
    case "FileText": return <FileText {...props} />;
    case "Layers": return <Layers {...props} />;
    case "LogOut": return <LogOut {...props} />;
    case "MessageSquare": return <MessageSquare {...props} />;
    case "Percent": return <Percent {...props} />;
    case "ShieldAlert": return <ShieldAlert {...props} />;
    case "Sliders": return <Sliders {...props} />;
    case "TrendingUp": return <TrendingUp {...props} />;
    case "Users": return <Users {...props} />;
    default: return <LayoutDashboard {...props} />;
  }
}

export default function Sidebar({ collapsed, setCollapsed }: SidebarProps) {
  const user = useAuthStore((state) => state.user);
  const visibleModules = moduleRoutes.filter(
    (item) => hasAnyRole(user, item.requiredRoles) && hasAnyPermission(user, item.requiredPermissions),
  );

  return (
    <aside
      className={`relative flex h-screen flex-col border-r-2 border-gold-champagne/35 bg-white transition-all duration-300 ${
        collapsed ? "w-[76px]" : "w-[292px]"
      }`}
    >
      <div className="flex h-[92px] items-center justify-between border-b-2 border-gold-champagne/25 bg-hero-bg px-5">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-[8px] bg-gold-champagne text-white shadow-[0_6px_16px_rgba(200,169,120,0.38)]">
            <Sparkles className="h-6 w-6 stroke-[2.3]" />
          </div>
          {!collapsed && (
            <div className="min-w-0 leading-tight">
              <div className="truncate text-[1.02rem] font-black tracking-wide text-text-title">
                WFAgent <span className="text-gold-champagne">ELITE</span>
              </div>
              <div className="mt-1 truncate font-mono text-[0.68rem] font-black uppercase tracking-[0.22em] text-text-body">
                Private Capital OS
              </div>
            </div>
          )}
        </div>
        {!collapsed && (
          <button
            className="ui-icon-button sidebar-toggle flex h-8 w-8 items-center justify-center rounded-full border-2 border-gold-champagne/35 bg-white text-gold-champagne shadow-sm hover:bg-khaki-light"
            onClick={() => setCollapsed(true)}
            type="button"
            title="收起侧边栏"
          >
            <ChevronLeft className="h-4 w-4 stroke-[2.4]" />
          </button>
        )}
      </div>

      <nav className="flex-1 overflow-y-auto px-3.5 py-5">
        <SidebarLink collapsed={collapsed} iconName={dashboardRoute.iconName} id={dashboardRoute.id} name={dashboardRoute.name} to="/" />
        {categories.map((category) => {
          const items = visibleModules.filter((item) => item.category === category);
          if (!items.length) return null;
          return (
            <section className="mt-6" key={category}>
              {!collapsed && (
                <div className="mb-2 border-l-2 border-gold-champagne/70 pl-2 text-[0.82rem] font-black text-text-body">
                  {categoryLabels[category]}
                </div>
              )}
              <div className="space-y-1.5">
                {items.map((item) => (
                  <SidebarLink key={item.id} collapsed={collapsed} iconName={item.iconName} id={item.id} name={item.name} to={`/${item.path}`} />
                ))}
              </div>
            </section>
          );
        })}
      </nav>

      {collapsed && (
        <button
          className="ui-icon-button sidebar-toggle absolute -right-3 top-7 flex h-7 w-7 items-center justify-center rounded-full border-2 border-gold-champagne/35 bg-white text-gold-champagne shadow-sm hover:bg-khaki-light"
          onClick={() => setCollapsed(false)}
          type="button"
          title="展开侧边栏"
        >
          <ChevronRight className="h-4 w-4 stroke-[2.4]" />
        </button>
      )}

      {!collapsed && (
        <div className="border-t-2 border-gold-champagne/30 bg-hero-bg px-5 py-4">
          <div className="flex items-center gap-2 font-mono text-[0.75rem] font-black uppercase text-gold-champagne">
            <span className="h-2.5 w-2.5 rounded-full bg-gold-champagne" />
            Secured Endpoint
          </div>
          <div className="mt-2 text-[0.78rem] font-bold text-text-sub">AUM Level: ¥5.2B Capital Pool</div>
        </div>
      )}
    </aside>
  );
}

interface SidebarLinkProps {
  collapsed: boolean;
  iconName: string;
  id: string;
  name: string;
  to: string;
}

function SidebarLink({ collapsed, iconName, id, name, to }: SidebarLinkProps) {
  return (
    <NavLink
      to={to}
      id={`sidebar-item-${id}`}
      end={to === "/"}
      className={({ isActive }) =>
        `sidebar-link relative flex h-11 items-center gap-3 rounded-[8px] px-3.5 text-[0.92rem] font-black transition-all duration-200 ${
          isActive
            ? "bg-manage-bg/65 text-text-title"
            : "text-text-sub hover:bg-khaki-light hover:text-text-title"
        }`
      }
      title={collapsed ? name : undefined}
    >
      {({ isActive }) => (
        <>
          {isActive && <span className="absolute left-0 top-2 bottom-2 w-1 rounded-r bg-gold-champagne" />}
          <Icon name={iconName} className={`h-5 w-5 shrink-0 ${isActive ? "text-gold-champagne" : "text-text-hint"}`} />
          {!collapsed && <span className="truncate">{name}</span>}
          {isActive && !collapsed && <span className="ml-auto h-2.5 w-2.5 rounded-full bg-gold-champagne/80" />}
        </>
      )}
    </NavLink>
  );
}
