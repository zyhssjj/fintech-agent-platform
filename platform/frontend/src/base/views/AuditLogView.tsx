import { BaseDataTable } from "@/base/components/BaseDataTable";

interface AuditRow {
  request_id: string;
  actor: string;
  action: string;
  resource: string;
  ip: string;
  created_at: string;
}

export function AuditLogView() {
  const rows: AuditRow[] = [];

  return (
    <section className="rounded-[8px] border border-slate-200 bg-white p-6">
      <h1 className="text-lg font-bold text-slate-950">操作日志</h1>
      <p className="mt-1 text-sm text-slate-500">基座保留审计日志查看页。后端开放查询接口后，只需在本页接入 `request.get` 即可展示真实日志。</p>
      <div className="mt-5">
        <BaseDataTable
          rows={rows}
          rowKey={(row) => row.request_id}
          columns={[
            { key: "request_id", title: "请求 ID", filterable: true },
            { key: "actor", title: "操作人", filterable: true },
            { key: "action", title: "动作", filterable: true },
            { key: "resource", title: "资源" },
            { key: "ip", title: "IP" },
            { key: "created_at", title: "时间", sortable: true },
          ]}
        />
      </div>
    </section>
  );
}
