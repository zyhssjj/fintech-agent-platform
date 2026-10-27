import React from "react";
import { Send } from "lucide-react";
import { request } from "@/base/api";
import { useUIStore } from "@/base/stores/uiStore";

export function NotificationCenterView() {
  const notify = useUIStore((state) => state.notify);
  const [form, setForm] = React.useState({ title: "", content: "", type: "system", recipients: "" });
  const [loading, setLoading] = React.useState(false);

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setLoading(true);
    try {
      await request.post("/agents/customer/notifications", {
        title: form.title,
        content: form.content,
        type: form.type,
        recipients: form.recipients.split(",").map((item) => item.trim()).filter(Boolean),
      });
      notify({ type: "success", title: "通知已提交" });
      setForm({ title: "", content: "", type: "system", recipients: "" });
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="rounded-[8px] border border-slate-200 bg-white p-6">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-[8px] bg-slate-100">
          <Send className="h-5 w-5 text-slate-600" />
        </div>
        <div>
          <h1 className="text-lg font-bold text-slate-950">系统通知中心</h1>
          <p className="text-sm text-slate-500">示例调用 `/agents/customer/notifications`，用于验证基座 API 封装。</p>
        </div>
      </div>
      <form className="mt-6 grid gap-3" onSubmit={handleSubmit}>
        <input className="h-11 rounded-[8px] border border-slate-200 px-3 text-sm outline-none focus:border-slate-500" placeholder="通知标题" value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} />
        <textarea className="min-h-28 rounded-[8px] border border-slate-200 px-3 py-2 text-sm outline-none focus:border-slate-500" placeholder="通知内容" value={form.content} onChange={(event) => setForm({ ...form, content: event.target.value })} />
        <input className="h-11 rounded-[8px] border border-slate-200 px-3 text-sm outline-none focus:border-slate-500" placeholder="接收人，多个用英文逗号分隔" value={form.recipients} onChange={(event) => setForm({ ...form, recipients: event.target.value })} />
        <button className="h-10 w-fit rounded-[8px] bg-slate-900 px-4 text-sm font-bold text-white disabled:opacity-60" disabled={loading} type="submit">{loading ? "提交中..." : "发送通知"}</button>
      </form>
    </section>
  );
}
