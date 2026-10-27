import { useState } from "react";
import { agentApi } from "@/module-agent/api";

const companies = [
  { code: "beijing_hangke", name: "北京航科", domains: ["基金层面", "项目层面", "公司层面"] },
  { code: "beijing_motian", name: "北京膜天", domains: ["基金层面", "项目层面", "公司层面"] },
  { code: "neimeng_juyuan", name: "内蒙聚元", domains: ["基金层面", "项目层面", "公司层面"] },
  { code: "hunan_hangtian", name: "湖南航天", domains: ["基金层面", "项目层面", "公司层面"] },
  { code: "hangmo_group", name: "航膜集团", domains: ["公司层面"] },
  { code: "future_intelligence", name: "未来智能", domains: ["公司层面"] },
];

export function KnowledgeView() {
  const [query, setQuery] = useState("");
  const [companyCode, setCompanyCode] = useState("beijing_hangke");
  const company = companies.find((item) => item.code === companyCode) ?? companies[0];
  const [selected, setSelected] = useState(company.domains);
  const [mode, setMode] = useState<"search" | "answer">("answer");
  const [result, setResult] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const submit = async () => {
    if (!query.trim()) return;
    setLoading(true); setError("");
    try {
      const data = mode === "answer"
        ? await agentApi.answerKnowledge({ query, company_code: companyCode, domain_paths: selected, top_k: 8 })
        : await agentApi.searchKnowledge({ query, company_code: companyCode, domain_paths: selected, top_k: 8 });
      setResult(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "知识库请求失败");
    } finally { setLoading(false); }
  };

  const sources = result?.sources ?? result?.data?.sources ?? [];
  const answer = result?.answer ?? result?.data?.answer;
  return <div className="module-page knowledge-page">
    <div className="page-header"><div><div className="eyebrow">KNOWLEDGE & COMPLIANCE</div><h1>知识合规库</h1><p>{company.name}业务资料检索与证据问答</p></div><span className="page-badge">服务端代理 · 证据可追溯</span></div>
    <section className="panel"><label className="knowledge-company"><span>公司知识域</span><select value={companyCode} onChange={(event) => { const next = companies.find((item) => item.code === event.target.value) ?? companies[0]; setCompanyCode(next.code); setSelected(next.domains); setResult(null); }}>{companies.map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}</select></label><div className="segmented-control"><button className={mode === "answer" ? "active" : ""} onClick={() => setMode("answer")}>智能回答</button><button className={mode === "search" ? "active" : ""} onClick={() => setMode("search")}>证据检索</button></div><textarea value={query} onChange={(event) => setQuery(event.target.value)} placeholder="例如：基金运营中需要保留哪些核心资料？" rows={4} /><div className="knowledge-toolbar"><div>{company.domains.map((domain) => <label key={domain}><input type="checkbox" checked={selected.includes(domain)} onChange={() => setSelected((items) => items.includes(domain) ? items.filter((item) => item !== domain) : [...items, domain])} /> {domain}</label>)}</div><button className="button primary" onClick={submit} disabled={loading || !query.trim()}>{loading ? "查询中..." : mode === "answer" ? "开始回答" : "检索证据"}</button></div></section>
    {error && <div className="panel error-panel">{error}</div>}
    {answer && <section className="panel"><h2>回答</h2><p className="knowledge-answer">{answer}</p></section>}
    {result && <section className="panel"><div className="panel-title"><h2>来源证据</h2><span>{sources.length} 条</span></div>{sources.length ? sources.map((source: any, index: number) => <article className="knowledge-source" key={`${source.source}-${index}`}><div><strong>{source.title || source.source || `证据 ${index + 1}`}</strong><small>{source.domain_path || ""} · {source.document_version || "当前版本"}</small></div><p>{source.content || "暂无片段"}</p></article>) : <p>未找到匹配资料。</p>}</section>}
  </div>;
}
