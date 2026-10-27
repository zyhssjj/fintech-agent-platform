import React from "react";
import { AnimatePresence, motion } from "motion/react";
import { BrowserRouter, Navigate, Route, Routes, useLocation, useOutlet } from "react-router-dom";
import Sidebar from "@/components/Sidebar";
import TopBar from "@/components/TopBar";
import { AppShell } from "@/base/layout/AppShell";
import { GlobalFeedback } from "@/base/components/GlobalFeedback";
import { TabView } from "@/base/components/TabView";
import { getModuleByPath } from "@/base/router/moduleRegistry";
import { appRoutes } from "@/base/router/routes";
import { useNavigationStore } from "@/base/stores/navigationStore";
import { useUIStore } from "@/base/stores/uiStore";

function AppLayout() {
  const location = useLocation();
  const outlet = useOutlet();
  const activeModule = getModuleByPath(location.pathname);
  const sidebarCollapsed = useUIStore((state) => state.sidebarCollapsed);
  const setSidebarCollapsed = useUIStore((state) => state.setSidebarCollapsed);
  const openTab = useNavigationStore((state) => state.openTab);

  React.useEffect(() => {
    openTab({ path: location.pathname, title: activeModule?.name ?? "未命名页面" });
  }, [activeModule?.name, location.pathname, openTab]);

  return (
    <AppShell
      sidebar={<Sidebar collapsed={sidebarCollapsed} setCollapsed={setSidebarCollapsed} />}
      topbar={<TopBar activeModuleName={activeModule?.name ?? "模块页面"} />}
      tabs={<TabView />}
    >
      <AnimatePresence mode="wait" initial={false}>
        <motion.div
          className="app-page-shell"
          key={location.pathname}
          initial={{ opacity: 0, y: 14, scale: 0.992 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: -8, scale: 0.996 }}
          transition={{ duration: 0.24, ease: [0.16, 1, 0.3, 1] }}
        >
          {outlet}
        </motion.div>
      </AnimatePresence>
    </AppShell>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <GlobalFeedback />
      <Routes>
        <Route path="/login" element={appRoutes.login} />
        <Route path="/register" element={appRoutes.register} />
        <Route path="/forgot-password" element={appRoutes.forgotPassword} />
        <Route path="/" element={<AppLayout />}>
          <Route index element={appRoutes.dashboard} />
          <Route path="dashboard" element={<Navigate to="/" replace />} />
          <Route path="profile" element={appRoutes.profile} />
          {appRoutes.modules.map((moduleRoute) => (
            <Route key={moduleRoute.path} path={moduleRoute.path} element={moduleRoute.element} />
          ))}
          <Route path="403" element={appRoutes.forbidden} />
          <Route path="500" element={appRoutes.serverError} />
          <Route path="*" element={appRoutes.notFound} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
