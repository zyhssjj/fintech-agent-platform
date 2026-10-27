import React, { useState } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import { motion } from "motion/react";
import { BadgeCheck, LockKeyhole, ShieldCheck, Sparkles, UserRound } from "lucide-react";
import {
  AuthError,
  AuthExperience,
  AuthField,
  AuthSubmitButton,
  PasswordInput,
} from "@/base/components/AuthExperience";
import { authApi } from "@/base/api";
import { ApiError } from "@/base/api/request";
import { useAuthStore } from "@/base/stores/authStore";
import { useUIStore } from "@/base/stores/uiStore";

interface LocationState {
  from?: { pathname?: string };
}

const demoAccounts = [
  { username: "admin", password: "Admin@123456", label: "系统管理员" },
  { username: "fund_manager", password: "Fund@123456", label: "基金经理" },
  { username: "invest_manager", password: "Invest@123456", label: "投资经理" },
  { username: "post_invest_manager", password: "PostInvest@123456", label: "投后经理" },
  { username: "risk_manager", password: "Risk@123456", label: "风控经理" },
  { username: "legal_manager", password: "Legal@123456", label: "法务经理" },
  { username: "finance_manager", password: "Finance@123456", label: "财务经理" },
  { username: "ops_manager", password: "Ops@123456", label: "运营经理" },
  { username: "reviewer", password: "Reviewer@123456", label: "复核人员" },
  { username: "lp_user", password: "LP@123456", label: "LP 投资者" },
  { username: "guest", password: "Guest@123456", label: "访客" },
];

export function LoginView() {
  const navigate = useNavigate();
  const location = useLocation();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const notify = useUIStore((state) => state.notify);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [selectedDemo, setSelectedDemo] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const from = (location.state as LocationState | null)?.from?.pathname ?? "/";

  if (isAuthenticated) return <Navigate to={from} replace />;

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    setLoading(true);
    try {
      await authApi.login({ username, password });
      notify({ type: "success", title: "登录成功" });
      navigate(from, { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "登录失败，请稍后重试");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthExperience
      eyebrow="WELCOME BACK"
      title="登录工作台"
      description="进入科技金融智能体系统底座，继续处理基金、投资、投后和风控协同。"
      icon={<Sparkles className="h-5 w-5 stroke-[2.3]" />}
    >
      <form className="auth-form" onSubmit={handleSubmit}>
        <AuthField
          label="用户名"
          prefix={<UserRound className="h-4 w-4" />}
          value={username}
          onChange={(event) => {
            setUsername(event.target.value);
            setSelectedDemo("");
          }}
          autoComplete="username"
          required
        />
        <PasswordInput
          label="密码"
          prefix={<LockKeyhole className="h-4 w-4" />}
          value={password}
          onChange={(event) => {
            setPassword(event.target.value);
            setSelectedDemo("");
          }}
          autoComplete="current-password"
          required
        />

        <div className="auth-demo">
          <div className="auth-demo-head">
            <span>快速体验</span>
            <small>选择一个角色自动填入账号</small>
          </div>
          <div className="auth-demo-grid">
            {demoAccounts.map((account, index) => (
              <motion.button
                className={selectedDemo === account.username ? "is-active" : undefined}
                key={account.username}
                onClick={() => {
                  setUsername(account.username);
                  setPassword(account.password);
                  setSelectedDemo(account.username);
                  setError("");
                }}
                type="button"
                title={`${account.username} / ${account.password}`}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.32 + index * 0.025, duration: 0.32 }}
                whileHover={{ y: -2 }}
                whileTap={{ scale: 0.98 }}
              >
                <BadgeCheck className="h-3.5 w-3.5" />
                {account.label}
              </motion.button>
            ))}
          </div>
        </div>

        {error && <AuthError message={error} />}

        <AuthSubmitButton loading={loading} icon={<ShieldCheck className="h-4 w-4" />}>
          {loading ? "正在登录..." : "安全登录"}
        </AuthSubmitButton>

        <div className="auth-links">
          <Link to="/register">注册账号</Link>
          <Link to="/forgot-password">忘记密码</Link>
        </div>
      </form>
    </AuthExperience>
  );
}
