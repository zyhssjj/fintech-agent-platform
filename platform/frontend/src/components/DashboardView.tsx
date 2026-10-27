import { Activity, ArrowUpRight, Compass, LogOut, Sparkles, TrendingUp } from "lucide-react";

const metrics = [
  { label: "管理规模 AUM", value: "52.4 亿" },
  { label: "综合 IRR 业绩", value: "31.4%" },
  { label: "实缴到位率", value: "85.3%" },
];

const businessCards = [
  { eyebrow: "FUNDRAISING / 募", title: "智能一期科技基金募资进度", value: "¥ 12.5 亿", meta: "已实缴 85.3% / 目标 15 亿元", footer: "配置募资计划", icon: TrendingUp },
  { eyebrow: "INVESTING / 投", title: "在审项目管线与立项", value: "4 个核心项目", meta: "累计出资 4.0 亿 / 占比 15.4%", footer: "项目漏斗中心", icon: Compass },
  { eyebrow: "MANAGING / 管", title: "投后企业增值赋能监测", value: "¥ 35.8 亿", meta: "最新公允市值 / 增值 42.1%", footer: "资产画像与赋能", icon: Activity },
  { eyebrow: "EXITING / 退", title: "退出回流及 DPI 产出", value: "¥ 8.6 亿", meta: "已实现回款 / DPI 0.24x", footer: "退出方案配置", icon: LogOut },
];

export default function DashboardView() {
  return (
    <div className="space-y-8">
      <section className="rounded-[18px] border-2 border-gold-champagne/35 bg-hero-bg px-8 py-10 shadow-[0_8px_28px_rgba(200,169,120,0.10)]">
        <div className="grid gap-8 lg:grid-cols-[1.1fr_1fr] lg:items-center">
          <div>
            <div className="inline-flex items-center gap-2 rounded-[4px] bg-gold-champagne/18 px-3 py-1 font-mono text-[0.78rem] font-black uppercase tracking-[0.18em] text-gold-champagne">
              WFAGENT Intelligent Workspace
              <span className="h-2.5 w-2.5 rounded-full bg-gold-champagne/80" />
            </div>
            <h1 className="mt-5 max-w-3xl text-[2.15rem] font-black leading-tight text-text-title">
              募投管退全业务链路数字化大屏
            </h1>
            <p className="mt-4 max-w-3xl text-[1rem] font-semibold leading-8 text-text-sub">
              依托极简轻奢层级设计，整合风控、估值、LP 协同及 AI 智能体分析等 PE/VC 核心业务模块。
            </p>
          </div>
          <div className="grid gap-7 border-gold-champagne/35 lg:border-l-2 lg:pl-10">
            <div className="grid gap-7 sm:grid-cols-2">
              {metrics.slice(0, 2).map((metric) => (
                <MetricBlock key={metric.label} {...metric} />
              ))}
            </div>
            <MetricBlock {...metrics[2]} />
          </div>
        </div>
      </section>

      <section className="grid gap-6 md:grid-cols-2 xl:grid-cols-4">
        {businessCards.map((card) => {
          const Icon = card.icon;
          return (
            <article
              className="dashboard-business-card min-h-[250px] rounded-[14px] border-2 border-gold-champagne/35 bg-hero-bg p-6 shadow-[0_8px_22px_rgba(200,169,120,0.08)]"
              key={card.title}
            >
              <div className="flex items-center justify-between">
                <div className="dashboard-business-card__eyebrow font-mono text-[0.76rem] font-black uppercase tracking-[0.18em] text-gold-champagne">{card.eyebrow}</div>
                <div className="dashboard-business-card__icon flex h-10 w-10 items-center justify-center rounded-full bg-white text-gold-champagne shadow-[0_4px_12px_rgba(0,0,0,0.12)]">
                  <Icon className="h-5 w-5 stroke-[2.2]" />
                </div>
              </div>
              <h2 className="dashboard-business-card__title mt-8 text-[1rem] font-black text-text-body">{card.title}</h2>
              <div className="dashboard-business-card__value mt-5 text-[2rem] font-black leading-none text-gold-champagne">{card.value}</div>
              <div className="dashboard-business-card__meta mt-5 inline-flex rounded-[5px] bg-white/70 px-3 py-1 font-mono text-[0.78rem] font-black text-gold-champagne">
                {card.meta}
              </div>
              <div className="dashboard-business-card__footer mt-6 border-t-2 border-gold-champagne/25 pt-5 text-[0.84rem] font-black text-gold-champagne">
                {card.footer} <ArrowUpRight className="inline h-4 w-4" />
              </div>
            </article>
          );
        })}
      </section>

      <section className="grid gap-7 xl:grid-cols-[1fr_420px]">
        <div>
          <div className="flex items-start justify-between">
            <div className="border-l-[6px] border-gold-champagne pl-4">
              <h2 className="text-[1.45rem] font-black text-text-title">智能一期科技产业基金管线</h2>
              <p className="mt-1 text-[0.92rem] font-bold text-text-sub">精选先进硬件、半导体、具身智能方向在审项目群组</p>
            </div>
            <button className="dashboard-link-button mt-2 border-b-2 border-gold-champagne pb-1 text-[0.9rem] font-black text-gold-champagne" type="button">
              全部项目 <ArrowUpRight className="inline h-4 w-4" />
            </button>
          </div>
          <div className="mt-6 rounded-[14px] border-2 border-gold-champagne/30 bg-white p-6">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <div className="flex items-center gap-3">
                  <h3 className="text-[1.1rem] font-black text-text-title">星际半导体 (Starlight Semi)</h3>
                  <span className="rounded-[5px] border border-gold-champagne/30 bg-hero-bg px-2.5 py-1 text-xs font-black text-gold-champagne">尽调决策</span>
                </div>
                <p className="mt-3 text-sm font-semibold text-text-sub">行业：先进制造 / 市场半导体 / 融资轮次：B</p>
              </div>
              <div className="grid grid-cols-2 gap-6 text-right">
                <div>
                  <div className="text-xs font-black text-text-sub">预期收益 IRR</div>
                  <div className="mt-1 text-2xl font-black text-gold-champagne">28.4%</div>
                </div>
                <div>
                  <div className="text-xs font-black text-text-sub">风险评级</div>
                  <div className="mt-1 text-sm font-black text-text-body">中等风险</div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <aside className="rounded-[14px] border-2 border-gold-champagne/35 bg-hero-bg p-6">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-[7px] bg-gold-champagne text-white">
              <Sparkles className="h-5 w-5" />
            </div>
            <div className="font-mono text-[0.78rem] font-black uppercase tracking-[0.12em] text-gold-champagne">WFAgent 智能协同建议</div>
          </div>
          <p className="mt-6 text-[1rem] font-semibold leading-8 text-text-body">
            分析显示，星际半导体本期研发里程碑验证已通过，但关键合规报告尚未入库。建议在投决会表决前，使用 AI 智能体报告工具自动抓取近期行业交叉数据。
          </p>
        </aside>
      </section>
    </div>
  );
}

function MetricBlock({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[0.85rem] font-black text-text-sub">{label}</div>
      <div className="mt-2 flex items-end gap-5">
        <div className="text-[2.15rem] font-black leading-none text-gold-champagne">{value}</div>
        <MetricSparkline />
      </div>
    </div>
  );
}

function MetricSparkline() {
  return (
    <div className="dashboard-sparkline" aria-hidden="true">
      <svg viewBox="0 0 128 48" role="presentation">
        <defs>
          <linearGradient id="dashboard-sparkline-fill" x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stopColor="#c8a978" stopOpacity="0.28" />
            <stop offset="100%" stopColor="#c8a978" stopOpacity="0" />
          </linearGradient>
        </defs>
        <path className="dashboard-sparkline-area" d="M4 38 L24 31 L44 34 L64 22 L84 25 L104 13 L112 9 L124 9 L124 46 L4 46 Z" />
        <path className="dashboard-sparkline-line" pathLength="1" d="M4 38 L24 31 L44 34 L64 22 L84 25 L104 13 L112 9 L124 9" />
        <circle className="dashboard-sparkline-pulse" cx="112" cy="9" r="10" />
        <circle className="dashboard-sparkline-dot" cx="112" cy="9" r="4" />
      </svg>
    </div>
  );
}
