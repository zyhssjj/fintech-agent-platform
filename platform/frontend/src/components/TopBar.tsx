import React from "react";
import { Bell, Clock, LogOut, ShieldCheck, User } from "lucide-react";
import { Link, useNavigate } from "react-router-dom";
import { authApi } from "@/base/api";
import { useAuthStore } from "@/base/stores/authStore";

interface TopBarProps {
  activeModuleName: string;
}

export default function TopBar({ activeModuleName }: TopBarProps) {
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const clearSession = useAuthStore((state) => state.clearSession);
  const [time, setTime] = React.useState(new Date());

  React.useEffect(() => {
    const timer = window.setInterval(() => setTime(new Date()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  async function handleLogout() {
    try {
      await authApi.logout();
    } finally {
      clearSession();
      navigate("/login", { replace: true });
    }
  }

  return (
    <header className="flex h-[92px] items-center justify-between border-b-2 border-gold-champagne/25 bg-white px-7">
      <div className="flex min-w-0 items-center gap-2 font-mono text-[0.9rem] font-black uppercase tracking-[0.08em]">
        <Link className="text-text-sub hover:text-text-title" to="/">WFAGENT OS</Link>
        <span className="text-gold-champagne">/</span>
        <span className="truncate font-sans text-[0.98rem] tracking-normal text-text-title">{activeModuleName}</span>
      </div>

      <div className="flex items-center gap-6">
        <div className="hidden items-center gap-2 text-[0.86rem] font-black text-text-body lg:flex">
          <ShieldCheck className="h-4 w-4 text-gold-champagne stroke-[2.2]" />
          <span>加密数据审计节点已就绪</span>
        </div>
        <div className="hidden h-10 w-px bg-gold-champagne/45 lg:block" />
        <div className="hidden items-center gap-2 font-mono text-[0.86rem] font-black text-text-body md:flex">
          <Clock className="h-4 w-4 text-gold-champagne stroke-[2.2]" />
          <span>{formatTime(time)} UTC</span>
        </div>
        <div className="h-10 w-px bg-gold-champagne/45" />
        <Link className="relative rounded-full p-2 text-text-body hover:bg-khaki-light" to="/system/notifications" title="通知中心">
          <Bell className="h-5 w-5 stroke-[2.2]" />
          <span className="absolute right-2 top-2 h-2 w-2 rounded-full bg-gold-champagne" />
        </Link>
        <button className="rounded-full p-2 text-text-body hover:bg-khaki-light" onClick={handleLogout} type="button" title="退出登录">
          <LogOut className="h-5 w-5 stroke-[2.2]" />
        </button>
        <Link
          className="topbar-profile-link flex items-center gap-3 rounded-full border-2 border-gold-champagne/35 bg-white py-1.5 pl-5 pr-2 shadow-[0_5px_16px_rgba(200,169,120,0.16)] hover:bg-khaki-light"
          to="/profile"
          title="个人中心"
        >
          <div className="hidden text-right leading-tight sm:block">
            <div className="text-[0.86rem] font-black text-text-title">{user?.real_name || "首席投资官 (CIO)"}</div>
            <div className="mt-0.5 max-w-[180px] truncate font-mono text-[0.68rem] font-black text-text-sub">
              {user?.email || user?.username || "mikasal1468@gmail.com"}
            </div>
          </div>
          <span className="flex h-10 w-10 items-center justify-center rounded-full bg-gold-champagne text-white">
            <User className="h-5 w-5 stroke-[2.2]" />
          </span>
        </Link>
      </div>
    </header>
  );
}

function formatTime(date: Date) {
  const yyyy = date.getFullYear();
  const mm = String(date.getMonth() + 1).padStart(2, "0");
  const dd = String(date.getDate()).padStart(2, "0");
  const hh = String(date.getHours()).padStart(2, "0");
  const min = String(date.getMinutes()).padStart(2, "0");
  const sec = String(date.getSeconds()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd} ${hh}:${min}:${sec}`;
}
