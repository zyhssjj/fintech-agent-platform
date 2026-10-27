import { X } from "lucide-react";
import { NavLink, useLocation, useNavigate } from "react-router-dom";
import { useNavigationStore } from "@/base/stores/navigationStore";

export function TabView() {
  const location = useLocation();
  const navigate = useNavigate();
  const tabs = useNavigationStore((state) => state.tabs);
  const closeTab = useNavigationStore((state) => state.closeTab);

  function handleClose(path: string) {
    const active = location.pathname === path;
    closeTab(path);
    if (active) {
      const next = tabs.find((tab) => tab.path !== path)?.path ?? "/";
      navigate(next, { replace: true });
    }
  }

  return (
    <div className="flex h-11 items-center gap-2 overflow-x-auto border-b border-slate-200 bg-white px-4">
      {tabs.map((tab) => (
        <NavLink
          className={({ isActive }) =>
            `tab-chip flex h-8 shrink-0 items-center gap-2 rounded-[8px] border px-3 text-sm font-semibold ${
              isActive ? "is-active border-slate-900 bg-slate-900 text-white" : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
            }`
          }
          key={tab.path}
          to={tab.path}
          end={tab.path === "/"}
        >
          <span>{tab.title}</span>
          {tab.path !== "/" && (
            <button className="rounded p-0.5 hover:bg-white/20" onClick={(event) => { event.preventDefault(); handleClose(tab.path); }} type="button" title="关闭标签">
              <X className="h-3.5 w-3.5" />
            </button>
          )}
        </NavLink>
      ))}
    </div>
  );
}
