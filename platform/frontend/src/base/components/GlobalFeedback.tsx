import { AlertCircle, CheckCircle2, Info, Loader2, X } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { useUIStore, type NoticeType } from "@/base/stores/uiStore";

const iconByType: Record<NoticeType, typeof Info> = {
  success: CheckCircle2,
  error: AlertCircle,
  warning: AlertCircle,
  info: Info,
};

export function GlobalFeedback() {
  const loading = useUIStore((state) => state.loadingCount > 0);
  const notices = useUIStore((state) => state.notices);
  const removeNotice = useUIStore((state) => state.removeNotice);

  return (
    <>
      <AnimatePresence>
        {loading && (
          <motion.div
            className="global-loading-toast"
            initial={{ opacity: 0, y: -12, x: "-50%", scale: 0.96 }}
            animate={{ opacity: 1, y: 0, x: "-50%", scale: 1 }}
            exit={{ opacity: 0, y: -10, x: "-50%", scale: 0.96 }}
            transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }}
          >
            <span className="fintech-loader fintech-loader--mini" aria-hidden="true" />
            <Loader2 className="h-4 w-4 animate-spin text-gold-champagne" />
            API 请求中
          </motion.div>
        )}
      </AnimatePresence>

      <div className="fixed right-4 top-4 z-50 flex w-[min(360px,calc(100vw-2rem))] flex-col gap-2">
        <AnimatePresence initial={false}>
          {notices.map((notice) => {
            const Icon = iconByType[notice.type];
            return (
              <motion.div
                key={notice.id}
                className="notice-card"
                initial={{ opacity: 0, x: 18, scale: 0.98 }}
                animate={{ opacity: 1, x: 0, scale: 1 }}
                exit={{ opacity: 0, x: 18, scale: 0.98 }}
                transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }}
              >
                <div className="flex items-start gap-3">
                  <Icon className="mt-0.5 h-5 w-5 text-gold-champagne" />
                  <div className="min-w-0 flex-1">
                    <div className="font-semibold text-slate-900">{notice.title}</div>
                    {notice.message && <div className="mt-1 text-sm text-slate-600">{notice.message}</div>}
                  </div>
                  <button className="ui-icon-button rounded p-1 text-slate-400 hover:bg-slate-100" onClick={() => removeNotice(notice.id)} type="button" title="关闭通知">
                    <X className="h-4 w-4" />
                  </button>
                </div>
              </motion.div>
            );
          })}
        </AnimatePresence>
      </div>
    </>
  );
}
