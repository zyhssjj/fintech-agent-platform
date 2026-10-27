import React from "react";
import { Link, useNavigate } from "react-router-dom";
import { AtSign, LockKeyhole, Phone, Send, UserPlus, UserRound } from "lucide-react";
import {
  AuthError,
  AuthExperience,
  AuthField,
  AuthSubmitButton,
  PasswordInput,
  PasswordStrength,
} from "@/base/components/AuthExperience";
import { authApi } from "@/base/api";
import { ApiError } from "@/base/api/request";
import { useUIStore } from "@/base/stores/uiStore";

export function RegisterView() {
  const navigate = useNavigate();
  const notify = useUIStore((state) => state.notify);
  const [form, setForm] = React.useState({
    username: "",
    real_name: "",
    email: "",
    phone: "",
    password: "",
    confirm_password: "",
  });
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState("");

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    if (form.password !== form.confirm_password) {
      setError("两次输入的密码不一致");
      return;
    }
    setLoading(true);
    try {
      await authApi.register(form);
      notify({ type: "success", title: "注册成功", message: "账号已创建，管理员可继续分配角色权限。" });
      navigate("/login", { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "注册失败，请检查信息后重试");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthExperience
      eyebrow="CREATE ACCOUNT"
      title="注册账号"
      description="创建基础账号后默认拥有访客权限，后续可由管理员分配业务角色。"
      icon={<UserPlus className="h-5 w-5" />}
      accent="teal"
    >
      <form className="auth-form" onSubmit={handleSubmit}>
        <div className="auth-grid">
          <AuthField
            label="用户名"
            prefix={<UserRound className="h-4 w-4" />}
            value={form.username}
            onChange={(event) => setForm({ ...form, username: event.target.value })}
            autoComplete="username"
            required
          />
          <AuthField
            label="真实姓名"
            prefix={<UserRound className="h-4 w-4" />}
            value={form.real_name}
            onChange={(event) => setForm({ ...form, real_name: event.target.value })}
            required
          />
          <AuthField
            className="auth-grid-full"
            label="邮箱"
            prefix={<AtSign className="h-4 w-4" />}
            type="email"
            value={form.email}
            onChange={(event) => setForm({ ...form, email: event.target.value })}
            autoComplete="email"
            required
          />
          <AuthField
            className="auth-grid-full"
            label="手机号"
            prefix={<Phone className="h-4 w-4" />}
            value={form.phone}
            onChange={(event) => setForm({ ...form, phone: event.target.value })}
            autoComplete="tel"
            hint="可选，用于后续通知和业务联系。"
          />
          <PasswordInput
            label="密码"
            prefix={<LockKeyhole className="h-4 w-4" />}
            value={form.password}
            onChange={(event) => setForm({ ...form, password: event.target.value })}
            autoComplete="new-password"
            minLength={8}
            required
          />
          <PasswordInput
            label="确认密码"
            prefix={<LockKeyhole className="h-4 w-4" />}
            value={form.confirm_password}
            onChange={(event) => setForm({ ...form, confirm_password: event.target.value })}
            autoComplete="new-password"
            minLength={8}
            required
          />
        </div>

        <PasswordStrength password={form.password} confirmPassword={form.confirm_password} />
        {error && <AuthError message={error} />}

        <AuthSubmitButton loading={loading} icon={<Send className="h-4 w-4" />}>
          {loading ? "正在提交..." : "提交注册"}
        </AuthSubmitButton>

        <Link className="auth-back-link" to="/login">返回登录</Link>
      </form>
    </AuthExperience>
  );
}
