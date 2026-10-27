import { ServerCrash } from "lucide-react";
import { Link } from "react-router-dom";

export function ServerErrorView() {
  return (
    <section className="flex min-h-[calc(100vh-10rem)] items-center justify-center">
      <div className="w-full max-w-md rounded-[8px] border border-slate-200 bg-white p-8 text-center shadow-sm">
        <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-slate-100 text-slate-600">
          <ServerCrash className="h-10 w-10" />
        </div>
        <div className="mt-6 text-sm font-bold uppercase tracking-widest text-slate-400">500</div>
        <h1 className="mt-2 text-xl font-bold text-slate-950">系统异常</h1>
        <p className="mt-3 text-sm leading-6 text-slate-600">请求后端服务失败或页面发生异常时，可跳转到该兜底页面。</p>
        <Link className="mt-6 inline-flex h-10 items-center rounded-[8px] bg-slate-900 px-4 text-sm font-semibold text-white" to="/">返回工作台</Link>
      </div>
    </section>
  );
}
