import React, { useId, useMemo, useState } from "react";
import { motion } from "motion/react";
import type { HTMLMotionProps } from "motion/react";
import {
  AlertCircle,
  CheckCircle2,
  Eye,
  EyeOff,
  ShieldCheck,
} from "lucide-react";

interface AuthExperienceProps {
  children: React.ReactNode;
  eyebrow: string;
  title: string;
  description: string;
  icon: React.ReactNode;
  accent?: "gold" | "teal" | "indigo";
}

type AuthFieldProps = Omit<React.InputHTMLAttributes<HTMLInputElement>, "prefix"> & {
  label: string;
  prefix?: React.ReactNode;
  hint?: string;
};

type AuthSubmitButtonProps = Omit<HTMLMotionProps<"button">, "children"> & {
  children: React.ReactNode;
  loading?: boolean;
  icon?: React.ReactNode;
};

const accentCopy = {
  gold: {
    tone: "auth-accent-gold",
  },
  teal: {
    tone: "auth-accent-teal",
  },
  indigo: {
    tone: "auth-accent-indigo",
  },
};

export function AuthExperience({
  children,
  eyebrow,
  title,
  description,
  icon,
  accent = "gold",
}: AuthExperienceProps) {
  const accentMeta = accentCopy[accent];

  return (
    <main className={`auth-page ${accentMeta.tone}`}>
      <motion.section
        className="auth-stage"
        aria-label={title}
        initial={{ opacity: 0, y: 18, scale: 0.985 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ duration: 0.68, ease: [0.16, 1, 0.3, 1] }}
      >
        <section className="auth-panel">
          <motion.div
            className="auth-brand"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.12, duration: 0.48, ease: [0.16, 1, 0.3, 1] }}
          >
            <div className="auth-brand-mark">{icon}</div>
            <div>
              <p className="auth-eyebrow">{eyebrow}</p>
              <p className="auth-brand-name">WFAgent ELITE</p>
            </div>
          </motion.div>

          <motion.header
            className="auth-heading"
            initial={{ opacity: 0, y: 14 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2, duration: 0.5, ease: [0.16, 1, 0.3, 1] }}
          >
            <h1>{title}</h1>
            <p>{description}</p>
          </motion.header>

          <motion.div
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.28, duration: 0.52, ease: [0.16, 1, 0.3, 1] }}
          >
            {children}
          </motion.div>
        </section>
      </motion.section>
    </main>
  );
}

export function AuthField({ label, prefix, hint, className = "", id, ...inputProps }: AuthFieldProps) {
  const fallbackId = useId();
  const inputId = id ?? fallbackId;

  return (
    <label className={`auth-field ${className}`} htmlFor={inputId}>
      <span className="auth-label">{label}</span>
      <span className="auth-input-shell">
        <span className="auth-input-focus-glow" aria-hidden="true" />
        {prefix && <span className="auth-input-icon">{prefix}</span>}
        <input id={inputId} className="auth-input" {...inputProps} />
      </span>
      {hint && <span className="auth-field-hint">{hint}</span>}
    </label>
  );
}

export function PasswordInput({ label, prefix, hint, className = "", id, ...inputProps }: AuthFieldProps) {
  const [visible, setVisible] = useState(false);
  const fallbackId = useId();
  const inputId = id ?? fallbackId;

  return (
    <label className={`auth-field ${className}`} htmlFor={inputId}>
      <span className="auth-label">{label}</span>
      <span className="auth-input-shell">
        <span className="auth-input-focus-glow" aria-hidden="true" />
        {prefix && <span className="auth-input-icon">{prefix}</span>}
        <input id={inputId} className="auth-input auth-input-with-action" type={visible ? "text" : "password"} {...inputProps} />
        <button
          aria-label={visible ? "隐藏密码" : "显示密码"}
          className="auth-input-action ui-icon-button"
          onClick={() => setVisible((current) => !current)}
          type="button"
        >
          {visible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
        </button>
      </span>
      {hint && <span className="auth-field-hint">{hint}</span>}
    </label>
  );
}

export function AuthSubmitButton({
  children,
  loading,
  icon,
  className = "",
  ...buttonProps
}: AuthSubmitButtonProps) {
  return (
    <motion.button
      className={`auth-submit ${className}`}
      disabled={loading || buttonProps.disabled}
      type="submit"
      {...buttonProps}
    >
      <span className="auth-submit-flow" aria-hidden="true" />
      <span className="auth-submit-glow" aria-hidden="true" />
      <span className="auth-submit-label">
        {loading ? <span className="fintech-loader fintech-loader--button" aria-hidden="true" /> : icon}
        <span>{children}</span>
      </span>
    </motion.button>
  );
}

export function AuthError({ message }: { message: string }) {
  return (
    <motion.div
      className="auth-error"
      initial={{ opacity: 0, y: -8 }}
      animate={{ opacity: 1, y: 0 }}
      role="alert"
    >
      <AlertCircle className="h-4 w-4" />
      <span>{message}</span>
    </motion.div>
  );
}

export function PasswordStrength({ password, confirmPassword }: { password: string; confirmPassword?: string }) {
  const result = useMemo(() => getPasswordStrength(password), [password]);
  const matches = Boolean(confirmPassword) && password === confirmPassword;

  return (
    <div className="auth-strength" aria-live="polite">
      <div className="auth-strength-bars">
        {[0, 1, 2, 3].map((item) => (
          <span key={item} className={item < result.score ? "is-active" : undefined} />
        ))}
      </div>
      <div className="auth-strength-copy">
        <span>{result.label}</span>
        {confirmPassword && (
          <span className={matches ? "is-match" : "is-muted"}>
            {matches ? <CheckCircle2 className="h-3.5 w-3.5" /> : <ShieldCheck className="h-3.5 w-3.5" />}
            {matches ? "两次密码一致" : "等待确认密码匹配"}
          </span>
        )}
      </div>
    </div>
  );
}

export function AuthStepIndicator({ current }: { current: "request" | "reset" }) {
  const steps = [
    { key: "request", label: "验证账号" },
    { key: "reset", label: "设置新密码" },
  ] as const;

  return (
    <div className="auth-steps" aria-label="找回密码步骤">
      {steps.map((step, index) => (
        <div className={current === step.key ? "is-active" : undefined} key={step.key}>
          <span>{index + 1}</span>
          <strong>{step.label}</strong>
        </div>
      ))}
    </div>
  );
}

function getPasswordStrength(password: string) {
  const score = [
    password.length >= 8,
    /[A-Z]/.test(password) && /[a-z]/.test(password),
    /\d/.test(password),
    /[^A-Za-z0-9]/.test(password),
  ].filter(Boolean).length;

  const labels = ["请输入至少 8 位密码", "强度较弱", "强度适中", "强度良好", "强度优秀"];
  return { score, label: labels[score] };
}
