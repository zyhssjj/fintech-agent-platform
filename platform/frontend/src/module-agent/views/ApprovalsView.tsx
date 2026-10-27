import React from "react";
import { authStore } from "@/base/stores/authStore";
import { communicationApi, type ApprovalItem } from "@/module-agent/api/communication";

type ApprovalAction = "approve" | "reject" | "return" | "remind";

const businessTypeLabelMap: Record<string, string> = {
  template_publish: "模板发布",
  document_send: "函件发送",
};

const statusLabelMap: Record<string, string> = {
  pending: "待处理",
  approved: "已通过",
  rejected: "已驳回",
  returned: "已退回",
};

const actionLabelMap: Record<string, string> = {
  submit: "发起审批",
  approve: "通过",
  reject: "驳回",
  return: "退回修改",
  remind: "催办",
  auto_remind: "系统自动催办",
};

function isActionableApproval(item: ApprovalItem, roles: string[]) {
  return item.status === "pending" && (roles.includes("SYS_ADMIN") || roles.includes(item.current_approver_role));
}

function labelOf(map: Record<string, string>, value: string) {
  return map[value] || value || "-";
}

export function ApprovalsView() {
  const roles = React.useMemo(() => authStore.getSnapshot().user?.roles ?? [], []);
  const [approvals, setApprovals] = React.useState<ApprovalItem[]>([]);
  const [selectedApprovalId, setSelectedApprovalId] = React.useState<string>("");
  const [comment, setComment] = React.useState("");
  const [submittingAction, setSubmittingAction] = React.useState<ApprovalAction | "">("");

  const selected = React.useMemo(
    () => approvals.find((item) => item.approval_id === selectedApprovalId) ?? approvals[0] ?? null,
    [approvals, selectedApprovalId],
  );

  const load = React.useCallback(async () => {
    const data = await communicationApi.listApprovals({ status: "pending" });
    const actionableList = data.list.filter((item) => isActionableApproval(item, roles));
    setApprovals(actionableList);
    setSelectedApprovalId((current) => {
      if (current && actionableList.some((item) => item.approval_id === current)) {
        return current;
      }
      return actionableList[0]?.approval_id ?? "";
    });
  }, [roles]);

  React.useEffect(() => {
    void load();
  }, [load]);

  async function handleAction(action: ApprovalAction) {
    if (!selected || submittingAction) return;
    setSubmittingAction(action);
    try {
      await communicationApi.actApproval(selected.approval_id, action, comment);
      setComment("");
      setApprovals((current) => {
        const next = current.filter((item) => item.approval_id !== selected.approval_id);
        setSelectedApprovalId(next[0]?.approval_id ?? "");
        return next;
      });
      await load();
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      if (message.includes("approval is already") || message.includes("lacks permission")) {
        setComment("");
        await load();
        return;
      }
      throw error;
    } finally {
      setSubmittingAction("");
    }
  }

  return (
    <section className="grid gap-4 lg:grid-cols-[0.95fr_1.05fr]">
      <section className="rounded-[24px] bg-white p-6 shadow-sm ring-1 ring-black/5">
        <h1 className="text-[1.5rem] font-black text-slate-900">内部协同审批</h1>
        <p className="mt-2 text-sm text-slate-500">这里只展示当前可由你处理的审批待办，处理完成后会自动从列表移除。</p>
        {!approvals.length ? (
          <p className="mt-5 text-sm text-slate-500">当前没有待处理审批。</p>
        ) : (
          <div className="mt-5 space-y-3">
            {approvals.map((item) => (
              <button
                key={item.approval_id}
                className={`block w-full rounded-2xl border p-4 text-left ${selected?.approval_id === item.approval_id ? "border-slate-900 bg-slate-900 text-white" : "border-slate-100 bg-white text-slate-800"}`}
                onClick={() => setSelectedApprovalId(item.approval_id)}
              >
                <div className="font-semibold">{item.title}</div>
                <div className={`mt-2 text-xs ${selected?.approval_id === item.approval_id ? "text-slate-200" : "text-slate-500"}`}>
                  {labelOf(businessTypeLabelMap, item.business_type)} / {labelOf(statusLabelMap, item.status)} / step {item.current_step_no}
                </div>
              </button>
            ))}
          </div>
        )}
      </section>

      <section className="rounded-[24px] bg-white p-6 shadow-sm ring-1 ring-black/5">
        <h2 className="text-lg font-black text-slate-900">审批详情</h2>
        {!selected ? (
          <p className="mt-4 text-sm text-slate-500">请选择一条待处理审批。</p>
        ) : (
          <div className="mt-4 space-y-4">
            <div className="grid gap-3 md:grid-cols-2">
              <Detail label="审批单号" value={selected.approval_id} />
              <Detail label="业务类型" value={labelOf(businessTypeLabelMap, selected.business_type)} />
              <Detail label="当前状态" value={labelOf(statusLabelMap, selected.status)} />
              <Detail label="当前角色" value={selected.current_approver_role || "-"} />
              <Detail label="申请人" value={selected.applicant_name || "-"} />
              <Detail label="截止时间" value={selected.deadline_at || "-"} />
            </div>
            <div className="rounded-2xl bg-slate-50 p-4">
              <div className="text-sm font-semibold text-slate-700">流转日志</div>
              <div className="mt-3 space-y-2">
                {selected.progress_log.map((log, index) => (
                  <div key={index} className="rounded-xl bg-white p-3 text-sm text-slate-600 ring-1 ring-slate-100">
                    {String(log.time ?? "")} / {String(log.operator_name ?? "-")} / {labelOf(actionLabelMap, String(log.action ?? ""))} / {String(log.comment ?? "")}
                  </div>
                ))}
              </div>
            </div>
            <label className="grid gap-2 text-sm font-semibold text-slate-700">
              <span>审批意见</span>
              <textarea
                className="rounded-2xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-slate-400 disabled:bg-slate-50"
                rows={4}
                value={comment}
                disabled={Boolean(submittingAction)}
                onChange={(event) => setComment(event.target.value)}
              />
            </label>
            <div className="flex flex-wrap gap-2">
              <ActionButton label="通过" colorClass="bg-emerald-600" disabled={Boolean(submittingAction)} active={submittingAction === "approve"} onClick={() => void handleAction("approve")} />
              <ActionButton label="驳回" colorClass="bg-rose-600" disabled={Boolean(submittingAction)} active={submittingAction === "reject"} onClick={() => void handleAction("reject")} />
              <ActionButton label="退回修改" colorClass="bg-amber-500" disabled={Boolean(submittingAction)} active={submittingAction === "return"} onClick={() => void handleAction("return")} />
              <ActionButton label="催办记录" colorClass="bg-slate-900" disabled={Boolean(submittingAction)} active={submittingAction === "remind"} onClick={() => void handleAction("remind")} />
            </div>
          </div>
        )}
      </section>
    </section>
  );
}

function ActionButton({
  label,
  colorClass,
  disabled,
  active,
  onClick,
}: {
  label: string;
  colorClass: string;
  disabled: boolean;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      className={`rounded-full px-4 py-2 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-slate-300 ${active ? "opacity-80" : ""} ${colorClass}`}
      disabled={disabled}
      onClick={onClick}
    >
      {active ? "处理中..." : label}
    </button>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-slate-50 p-4">
      <div className="text-xs font-bold uppercase tracking-[0.18em] text-slate-400">{label}</div>
      <div className="mt-2 text-sm font-semibold text-slate-900">{value}</div>
    </div>
  );
}

export default ApprovalsView;
