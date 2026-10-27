import React from "react";
import { KeyRound, Save } from "lucide-react";
import { request } from "@/base/api";

type APIConfig = {
  provider: string;
  base_url: string;
  api_key: string;
  model: string;
  timeout_ms: string;
  api_key_configured: boolean;
};

export function SystemConfigView() {
  const [form, setForm] = React.useState<APIConfig>({ provider: "deepseek", base_url: "", api_key: "", model: "deepseek-v4-flash", timeout_ms: "60000", api_key_configured: false });
  const [message, setMessage] = React.useState("");
  const [saving, setSaving] = React.useState(false);

  React.useEffect(() => {
    void request.get<Omit<APIConfig, "api_key">>("/system/api-config").then((value) => setForm((prev) => ({ ...prev, ...value }))).catch((error) => setMessage(error instanceof Error ? error.message : "加载 API 配置失败"));
  }, []);

  async function save() {
    setSaving(true);
    try {
      const result = await request.put<{ restart_required: boolean }>("/system/api-config", form);
      setForm((prev) => ({ ...prev, api_key: "", api_key_configured: prev.api_key_configured || Boolean(form.api_key) }));
      setMessage(result.restart_required ? "配置已写入 ai/.env。请重启 AI 服务后生效。" : "配置已保存");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存 API 配置失败");
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="max-w-3xl rounded-[8px] border border-slate-200 bg-white p-6">
      <div className="flex items-center gap-3"><KeyRound className="text-slate-700" /><div><h1 className="text-lg font-bold text-slate-950">API 配置</h1><p className="mt-1 text-sm text-slate-500">配置 AI 服务地址、模型与访问密钥。</p></div></div>
      <div className="mt-6 grid gap-4 md:grid-cols-2">
        <Field label="服务提供方" value={form.provider} onChange={(value) => setForm((prev) => ({ ...prev, provider: value }))} />
        <Field label="模型" value={form.model} onChange={(value) => setForm((prev) => ({ ...prev, model: value }))} />
        <div className="md:col-span-2"><Field label="API Base URL" value={form.base_url} onChange={(value) => setForm((prev) => ({ ...prev, base_url: value }))} /></div>
        <Field label="请求超时（毫秒）" type="number" value={form.timeout_ms} onChange={(value) => setForm((prev) => ({ ...prev, timeout_ms: value }))} />
        <Field label={form.api_key_configured ? "API Key（已配置，留空则不修改）" : "API Key"} type="password" value={form.api_key} onChange={(value) => setForm((prev) => ({ ...prev, api_key: value }))} />
      </div>
      <div className="mt-6 flex items-center gap-3"><button className="inline-flex items-center gap-2 rounded-full bg-slate-900 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50" disabled={saving} onClick={() => void save()}><Save size={16} />{saving ? "保存中..." : "保存 API 配置"}</button>{message ? <span className="text-sm text-slate-600">{message}</span> : null}</div>
    </section>
  );
}

function Field({ label, value, onChange, type = "text" }: { label: string; value: string; onChange: (value: string) => void; type?: string }) {
  return <label className="grid gap-2 text-sm font-semibold text-slate-700"><span>{label}</span><input className="h-11 rounded-[8px] border border-slate-200 px-3 text-sm outline-none focus:border-slate-500" type={type} value={value} onChange={(event) => onChange(event.target.value)} /></label>;
}
