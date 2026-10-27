import React from "react";
import { motion } from "motion/react";
import { KeyRound, LockKeyhole, Save, ShieldCheck, UserRound } from "lucide-react";
import { authApi } from "@/base/api";
import { ApiError } from "@/base/api/request";
import {
  AuthError,
  AuthSubmitButton,
  PasswordInput,
  PasswordStrength,
} from "@/base/components/AuthExperience";
import { useAuthStore } from "@/base/stores/authStore";
import { useUIStore } from "@/base/stores/uiStore";

export function ProfileView() {
  const user = useAuthStore((state) => state.user);
  const accessToken = useAuthStore((state) => state.accessToken);
  const setUser = useAuthStore((state) => state.setUser);
  const notify = useUIStore((state) => state.notify);
  const [passwordForm, setPasswordForm] = React.useState({ old_password: "", new_password: "", confirm_password: "" });
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState("");

  React.useEffect(() => {
    if (!accessToken) return;
    if (user) return;
    void authApi.me().then(setUser).catch(() => undefined);
  }, [accessToken, setUser, user]);

  async function handleChangePassword(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    if (passwordForm.new_password !== passwordForm.confirm_password) {
      setError("两次输入的新密码不一致");
      return;
    }
    setLoading(true);
    try {
      await authApi.changePassword(passwordForm);
      notify({ type: "success", title: "密码已修改", message: "下次登录请使用新密码。" });
      setPasswordForm({ old_password: "", new_password: "", confirm_password: "" });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "密码修改失败，请检查旧密码后重试");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="grid gap-5 lg:grid-cols-[1fr_430px]">
      <motion.section
        className="rounded-[8px] border border-slate-200 bg-white p-6 shadow-sm"
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.42, ease: [0.16, 1, 0.3, 1] }}
      >
        <div className="flex items-center gap-3">
          <div className="flex h-11 w-11 items-center justify-center rounded-[8px] bg-slate-100">
            <UserRound className="h-5 w-5 text-slate-600" />
          </div>
          <div>
            <h1 className="text-lg font-bold text-slate-950">个人中心</h1>
            <p className="text-sm text-slate-500">当前登录用户、角色与权限信息。</p>
          </div>
        </div>
        <dl className="mt-6 grid gap-4 sm:grid-cols-2">
          <Info label="用户名" value={user?.username} />
          <Info label="真实姓名" value={user?.real_name} />
          <Info label="邮箱" value={user?.email} />
          <Info label="部门" value={user?.departments?.join(" / ")} />
          <Info label="角色" value={user?.roles?.join(" / ")} wide />
          <Info label="权限" value={user?.permissions?.join(" / ")} wide />
        </dl>
      </motion.section>

      <motion.form
        className="profile-password-panel"
        onSubmit={handleChangePassword}
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.08, duration: 0.42, ease: [0.16, 1, 0.3, 1] }}
      >
        <div className="profile-password-panel__header">
          <div>
            <KeyRound className="h-5 w-5" />
          </div>
          <span>
            <strong>修改密码</strong>
            <small>修改成功后，新密码立即生效。</small>
          </span>
        </div>

        <div className="auth-form">
          <PasswordInput
            label="旧密码"
            prefix={<LockKeyhole className="h-4 w-4" />}
            value={passwordForm.old_password}
            onChange={(event) => setPasswordForm({ ...passwordForm, old_password: event.target.value })}
            autoComplete="current-password"
            required
          />
          <PasswordInput
            label="新密码"
            prefix={<ShieldCheck className="h-4 w-4" />}
            value={passwordForm.new_password}
            onChange={(event) => setPasswordForm({ ...passwordForm, new_password: event.target.value })}
            autoComplete="new-password"
            minLength={8}
            required
          />
          <PasswordInput
            label="确认新密码"
            prefix={<ShieldCheck className="h-4 w-4" />}
            value={passwordForm.confirm_password}
            onChange={(event) => setPasswordForm({ ...passwordForm, confirm_password: event.target.value })}
            autoComplete="new-password"
            minLength={8}
            required
          />
          <PasswordStrength password={passwordForm.new_password} confirmPassword={passwordForm.confirm_password} />
          {error && <AuthError message={error} />}
          <AuthSubmitButton className="profile-password-panel__submit" loading={loading} icon={<Save className="h-4 w-4" />}>
            {loading ? "正在提交..." : "保存密码"}
          </AuthSubmitButton>
        </div>
      </motion.form>
    </div>
  );
}

function Info({ label, value, wide = false }: { label: string; value?: string; wide?: boolean }) {
  return (
    <div className={wide ? "sm:col-span-2" : undefined}>
      <dt className="text-xs font-semibold uppercase text-slate-400">{label}</dt>
      <dd className="mt-1 rounded-[8px] bg-slate-50 px-3 py-2 text-sm text-slate-700">{value || "-"}</dd>
    </div>
  );
}
