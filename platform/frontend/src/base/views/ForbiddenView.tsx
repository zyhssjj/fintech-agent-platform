import { ShieldX } from "lucide-react";
import { Link } from "react-router-dom";

export function ForbiddenView() {
  return (
    <ErrorState code="403" title="无权访问" description="当前账号没有访问该页面所需的角色或权限。" icon={<ShieldX className="h-10 w-10" />} />
  );
}

function ErrorState({ code, title, description, icon }: { code: string; title: string; description: string; icon: React.ReactNode }) {
  return (
    <section className="flex min-h-[calc(100vh-10rem)] items-center justify-center">
      <div className="w-full max-w-md rounded-[8px] border border-slate-200 bg-white p-8 text-center shadow-sm">
        <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-slate-100 text-slate-600">{icon}</div>
        <div className="mt-6 text-sm font-bold uppercase tracking-widest text-slate-400">{code}</div>
        <h1 className="mt-2 text-xl font-bold text-slate-950">{title}</h1>
        <p className="mt-3 text-sm leading-6 text-slate-600">{description}</p>
        <Link className="mt-6 inline-flex h-10 items-center rounded-[8px] bg-slate-900 px-4 text-sm font-semibold text-white" to="/">返回工作台</Link>
      </div>
    </section>
  );
}
