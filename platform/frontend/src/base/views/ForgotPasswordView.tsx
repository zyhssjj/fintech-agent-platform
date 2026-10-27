import React from "react";
import { Link, useNavigate } from "react-router-dom";
import { AnimatePresence, motion } from "motion/react";
import { KeyRound, LockKeyhole, MailCheck, RotateCcw, Send, UserRound } from "lucide-react";
import {
  AuthError,
  AuthExperience,
  AuthField,
  AuthStepIndicator,
  AuthSubmitButton,
  PasswordInput,
  PasswordStrength,
} from "@/base/components/AuthExperience";
import { authApi } from "@/base/api";
import { ApiError } from "@/base/api/request";
import { useUIStore } from "@/base/stores/uiStore";

type Step = "request" | "reset";

export function ForgotPasswordView() {
  const navigate = useNavigate();
  const notify = useUIStore((state) => state.notify);
  const [step, setStep] = React.useState<Step>("request");
  const [account, setAccount] = React.useState("");
  const [resetToken, setResetToken] = React.useState("");
  const [passwordForm, setPasswordForm] = React.useState({ new_password: "", confirm_password: "" });
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState("");

  async function requestReset(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    setLoading(true);
    try {
      const result = await authApi.forgotPassword({ account });
      setResetToken(result.reset_token ?? "");
      setStep("reset");
      notify({ type: "success", title: "重置凭证已生成", message: "请继续设置新的登录密码。" });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "无法发起重置流程，请稍后重试");
    } finally {
      setLoading(false);
    }
  }

  async function resetPassword(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    if (passwordForm.new_password !== passwordForm.confirm_password) {
      setError("两次输入的新密码不一致");
      return;
    }
    setLoading(true);
    try {
      await authApi.resetPassword({
        reset_token: resetToken,
        new_password: passwordForm.new_password,
        confirm_password: passwordForm.confirm_password,
      });
      notify({ type: "success", title: "密码已重置", message: "请使用新密码登录。" });
      navigate("/login", { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "密码重置失败，请检查凭证后重试");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthExperience
      eyebrow="ACCOUNT RECOVERY"
      title="找回密码"
      description="验证账号并设置新的登录密码，整个过程会保留统一的安全校验和审计链路。"
      icon={step === "request" ? <MailCheck className="h-5 w-5" /> : <KeyRound className="h-5 w-5" />}
      accent="indigo"
    >
      <AuthStepIndicator current={step} />

      <AnimatePresence mode="wait">
        {step === "request" ? (
          <motion.form
            className="auth-form"
            key="request"
            onSubmit={requestReset}
            initial={{ opacity: 0, x: -18 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 18 }}
            transition={{ duration: 0.28 }}
          >
            <AuthField
              label="账号或邮箱"
              prefix={<UserRound className="h-4 w-4" />}
              placeholder="请输入账号或注册邮箱"
              value={account}
              onChange={(event) => setAccount(event.target.value)}
              autoComplete="username"
              required
            />
            {error && <AuthError message={error} />}
            <AuthSubmitButton loading={loading} icon={<Send className="h-4 w-4" />}>
              {loading ? "正在提交..." : "获取重置凭证"}
            </AuthSubmitButton>
            <Link className="auth-back-link" to="/login">返回登录</Link>
          </motion.form>
        ) : (
          <motion.form
            className="auth-form"
            key="reset"
            onSubmit={resetPassword}
            initial={{ opacity: 0, x: 18 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -18 }}
            transition={{ duration: 0.28 }}
          >
            <AuthField
              label="重置凭证"
              prefix={<KeyRound className="h-4 w-4" />}
              value={resetToken}
              onChange={(event) => setResetToken(event.target.value)}
              required
            />
            <PasswordInput
              label="新密码"
              prefix={<LockKeyhole className="h-4 w-4" />}
              value={passwordForm.new_password}
              onChange={(event) => setPasswordForm({ ...passwordForm, new_password: event.target.value })}
              autoComplete="new-password"
              minLength={8}
              required
            />
            <PasswordInput
              label="确认新密码"
              prefix={<LockKeyhole className="h-4 w-4" />}
              value={passwordForm.confirm_password}
              onChange={(event) => setPasswordForm({ ...passwordForm, confirm_password: event.target.value })}
              autoComplete="new-password"
              minLength={8}
              required
            />
            <PasswordStrength password={passwordForm.new_password} confirmPassword={passwordForm.confirm_password} />
            {error && <AuthError message={error} />}
            <AuthSubmitButton loading={loading} icon={<KeyRound className="h-4 w-4" />}>
              {loading ? "正在提交..." : "重置密码"}
            </AuthSubmitButton>
            <button className="auth-text-button" type="button" onClick={() => setStep("request")}>
              <RotateCcw className="h-3.5 w-3.5" />
              重新获取凭证
            </button>
          </motion.form>
        )}
      </AnimatePresence>
    </AuthExperience>
  );
}
