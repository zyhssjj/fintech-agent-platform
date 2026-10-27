const $ = (selector, root = document) => root.querySelector(selector);
const $$ = (selector, root = document) => [...root.querySelectorAll(selector)];

const state = {
  view: "overview",
  companyCode: "beijing_hangke",
  companies: [],
  searchMode: "retrieve",
  documents: [],
  documentDomain: "all",
  ingestionDomain: "",
  ingestionSelection: ["", "", ""],
  searchSelection: ["", "", ""],
  documentSelection: ["", "", ""],
  ingestionBusy: false,
  ingestionMode: "standard",
  files: [],
  agentScopePaths: new Set(),
  users: [],
  credentials: [],
  documentDetail: { id: "", page: 1, query: "", fullText: false },
};

function currentCompany() { return state.companies.find((item) => item.code === state.companyCode); }

const viewNames = {
  overview: "知识总览",
  search: "智能检索",
  documents: "文档中心",
  ingestion: "知识入库",
  users: "用户管理",
  agents: "智能体配置",
  models: "模型接入",
};

function escapeHtml(value = "") {
  return String(value).replace(/[&<>'"]/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[char]));
}

async function api(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  if (options.body && !(options.body instanceof FormData)) headers["Content-Type"] = "application/json";
  const response = await fetch(path, { ...options, headers });
  if (!response.ok) {
    let detail = response.statusText || "请求失败";
    try { detail = (await response.json()).detail || detail; } catch (_) {}
    throw new Error(detail);
  }
  return response.status === 204 ? {} : response.json();
}

let toastTimer;
function showToast(message) {
  const toast = $("#toast");
  toast.textContent = message;
  toast.classList.add("show");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toast.classList.remove("show"), 2800);
}

function setView(name, updateHash = true) {
  if (!viewNames[name]) name = "overview";
  state.view = name;
  $$(".view").forEach((view) => view.classList.toggle("active", view.id === `view-${name}`));
  $$(".nav-item").forEach((item) => item.classList.toggle("active", item.dataset.view === name));
  $("#current-page").textContent = viewNames[name];
  $("#sidebar").classList.remove("open");
  if (updateHash) history.replaceState(null, "", `#${name}`);
  if (name === "overview") loadOverview();
  if (name === "documents") loadDocuments();
  if (name === "users") loadUsers();
  if (name === "agents") loadAgents();
  if (name === "models") loadModels();
  if (name === "search") setTimeout(() => $("#search-query").focus(), 100);
  window.scrollTo({ top: 0, behavior: "smooth" });
}

$$('[data-view]').forEach((button) => button.addEventListener("click", () => setView(button.dataset.view)));
$$('[data-go]').forEach((button) => button.addEventListener("click", () => setView(button.dataset.go)));
$(".brand").addEventListener("click", (event) => { event.preventDefault(); setView("overview"); });
$("#menu-toggle").addEventListener("click", () => $("#sidebar").classList.toggle("open"));
window.addEventListener("hashchange", () => setView(location.hash.slice(1), false));
window.addEventListener("keydown", (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
    event.preventDefault(); setView("search");
  }
});

function readableTime(value) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" }).format(date);
}

async function loadOverview() {
  const companyCode = state.companyCode;
  try {
    const [overview, agents] = await Promise.all([api(`/api/overview?company_code=${encodeURIComponent(companyCode)}`), api("/api/credentials")]);
    if (state.companyCode !== companyCode) return;
    $("#stat-documents").textContent = overview.document_count ?? 0;
    $("#stat-chunks").textContent = overview.chunk_count ?? 0;
    $("#stat-agents").textContent = (agents.clients || []).length;
    $("#overview-time").textContent = `刚刚同步 · ${new Date().toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" })}`;
    const recent = [
      ...(overview.finance_docs || []).map((item) => ({ collection: item.domain_path || "科技金融", title: item.title || item.source, detail: item.source || "业务文档" })),
    ].slice(0, 5);
    $("#recent-list").innerHTML = recent.length ? recent.map((item) => `<div class="recent-item"><span>${item.collection}</span><div><b>${escapeHtml(item.title)}</b><small>${escapeHtml(item.detail)}</small></div><time>已入库</time></div>`).join("") : '<div class="table-empty">暂无已入库知识</div>';
  } catch (error) {
    if (state.companyCode !== companyCode) return;
    $("#recent-list").innerHTML = `<div class="table-empty">概览加载失败：${escapeHtml(error.message)}</div>`;
  }
}

$$('#search-mode button').forEach((button) => button.addEventListener("click", () => {
  state.searchMode = button.dataset.mode;
  $$('#search-mode button').forEach((item) => item.classList.toggle("active", item === button));
  $("#search-submit span").textContent = state.searchMode === "answer" ? "回答" : "检索";
}));

async function executeSearch() {
  const companyCode = state.companyCode;
  const query = $("#search-query").value.trim();
  const scope = CatalogControls.prefix(state.searchSelection);
  const domainPaths = scope ? [scope] : [];
  if (!query) return showToast("请先输入一个问题");
  const button = $("#search-submit");
  button.disabled = true;
  $("#result-summary").textContent = "正在执行混合检索与相关性重排…";
  $("#result-time").textContent = "";
  $("#search-results").className = "search-results";
  $("#search-results").innerHTML = '<div class="skeleton-row"></div><div class="skeleton-row"></div><div class="skeleton-row"></div>';
  try {
    const data = await api("/api/search", { method: "POST", body: JSON.stringify({ query, company_code: companyCode, domain_paths: domainPaths, mode: state.searchMode, top_k: 8 }) });
    if (state.companyCode !== companyCode || CatalogControls.prefix(state.searchSelection) !== scope) return;
    const sources = data.sources || [];
    const retrieval = data.retrieval || [];
    const listWithoutAnchor = retrieval.length && retrieval.every((item) => item.list_query && !item.section_anchor_found);
    $("#result-summary").textContent = listWithoutAnchor
      ? `未找到可完整列举的对应章节 · 展示 ${sources.length} 条相关证据`
      : `命中 ${sources.length} 条证据 · 路由至 ${(data.routed_to || []).map((item) => item.toUpperCase()).join("、") || "—"}`;
    $("#result-time").textContent = `${data.timing?.total_ms || 0} ms`;
    const answer = data.answer ? `<article class="answer-card"><strong>智能回答</strong>\n${escapeHtml(data.answer)}</article>` : "";
    const cards = sources.map((item, index) => `<article class="evidence-card"><div class="evidence-head"><div><span class="domain-pill">${escapeHtml(currentCompany()?.name || "")} / ${escapeHtml(item.domain_path || "科技金融")}</span><b>${index + 1}. ${escapeHtml(item.title || item.source || "未命名知识")}</b></div><span>${item.score ? `相关度 ${Number(item.score).toFixed(3)}` : "原文证据"}</span></div><p>${escapeHtml(item.content || "暂无正文")}</p><small>${escapeHtml(item.source || "")} · ${escapeHtml(item.document_version || "当前版本")}</small></article>`).join("");
    $("#search-results").innerHTML = answer + (cards || '<div class="table-empty">当前知识域未找到相关证据，请调整问题表述。</div>');
    if (retrieval.some((item) => item.list_query)) loadCompleteList(query, domainPaths);
  } catch (error) {
    if (state.companyCode !== companyCode || CatalogControls.prefix(state.searchSelection) !== scope) return;
    $("#result-summary").textContent = "检索未完成";
    $("#search-results").innerHTML = `<div class="table-empty">${escapeHtml(error.message)}</div>`;
  } finally { button.disabled = false; }
}

async function loadCompleteList(query, domainPaths) {
  const companyCode = state.companyCode;
  const scope = CatalogControls.prefix(state.searchSelection);
  try {
    const data = await api("/api/list", { method: "POST", body: JSON.stringify({ query, company_code: state.companyCode, domain_paths: domainPaths, page: 1, page_size: 50 }) });
    if (state.companyCode !== companyCode || CatalogControls.prefix(state.searchSelection) !== scope) return;
    if (!data.available) return;
    const header = data.table_header ? `<p class="complete-list-header">${escapeHtml(data.table_header)}</p>` : "";
    const rows = data.rows.map((row) => `<pre>${escapeHtml(row)}</pre>`).join("");
    const section = `<article class="complete-list-card"><div class="evidence-head"><div><span class="domain-pill">完整清单</span><b>${escapeHtml(data.section_title || "命中章节")}</b></div><span>共 ${data.total} 条 · 第 ${data.page}/${data.total_pages} 页</span></div>${header}<div class="complete-list-rows">${rows}</div><small>系统直接读取结构化原文行，不由 AI 生成。</small></article>`;
    $("#search-results").insertAdjacentHTML("afterbegin", section);
  } catch (_) {
    // Evidence retrieval remains usable when a matching chapter has no table rows.
  }
}

$("#search-submit").addEventListener("click", executeSearch);
$("#search-query").addEventListener("keydown", (event) => { if (event.key === "Enter" && !event.shiftKey) { event.preventDefault(); executeSearch(); } });

async function loadDocuments() {
  const companyCode = state.companyCode;
  const scope = CatalogControls.prefix(state.documentSelection);
  try {
    const params = new URLSearchParams({ company_code: companyCode });
    if (scope) params.append("domain_paths", scope);
    const data = await api(`/api/documents?${params}`);
    if (state.companyCode !== companyCode || CatalogControls.prefix(state.documentSelection) !== scope) return;
    state.documents = data.documents || [];
    renderDocuments();
  } catch (error) { if (state.companyCode === companyCode && CatalogControls.prefix(state.documentSelection) === scope) $("#document-list").innerHTML = `<tr><td colspan="6" class="table-empty">${escapeHtml(error.message)}</td></tr>`; }
}

function renderDocuments() {
  const query = $("#document-filter").value.trim().toLowerCase();
  const items = state.documents.filter((item) => (state.documentDomain === "all" || (item.domain_path || "").startsWith(state.documentDomain)) && `${item.name} ${item.source} ${item.domain_path}`.toLowerCase().includes(query));
  $("#document-list").innerHTML = items.length ? items.map((item) => `<tr><td><strong>${escapeHtml(item.name)}</strong><small>${escapeHtml(item.source || "业务文档")}</small></td><td><span class="domain-pill">${escapeHtml(item.domain_path || "未分类")}</span></td><td>${item.units} 个检索分块</td><td><span class="status-pill">可检索</span></td><td>${readableTime(item.updated_at)}</td><td><button class="table-action" data-document-detail="${escapeHtml(item.id)}" type="button">查看详情</button><button class="table-action danger" data-document-delete="${escapeHtml(item.id)}" type="button">删除</button></td></tr>`).join("") : '<tr><td colspan="6" class="table-empty">暂无匹配文档</td></tr>';
  $$('[data-document-detail]').forEach((button) => button.addEventListener("click", () => openDocumentDetail(button.dataset.documentDetail)));
  $$('[data-document-delete]').forEach((button) => button.addEventListener("click", () => deleteDocument(button.dataset.documentDelete)));
}

function aclLabel(acl) { return acl === "all" ? "当前平台已授权用户" : (acl || "未设置"); }

async function openDocumentDetail(documentId, page = 1, query = state.documentDetail.query, fullText = false) {
  const companyCode = state.companyCode;
  if (!documentId) return;
  state.documentDetail = { id: documentId, page, query, fullText };
  const dialog = $("#document-detail-dialog");
  if (!dialog.open) dialog.showModal();
  $("#document-detail-title").textContent = "正在读取文档…";
  $("#document-detail-chunks").innerHTML = '<div class="skeleton-row"></div><div class="skeleton-row"></div>';
  try {
    const params = new URLSearchParams({ company_code: state.companyCode, page: String(page), page_size: "8" });
    if (query) params.set("query", query);
    if (fullText) params.set("full_text", "true");
    const data = await api(`/api/documents/${encodeURIComponent(documentId)}/detail?${params}`);
    if (state.companyCode !== companyCode || state.documentDetail.id !== documentId) return;
    const doc = data.document;
    state.documentDetail = { id: documentId, page: data.page, query: data.query || "", fullText };
    $("#document-detail-title").textContent = doc.title || "未命名文档";
    $("#document-detail-subtitle").textContent = `${currentCompany()?.name || "科技金融"} · ${doc.domain_path || "未分类"} · 已入库自然语言分块审核`;
    $("#document-detail-meta").innerHTML = [
      ["原始文件", doc.source || "—"], ["解析方式", doc.document_type || doc.parser || "—"],
      ["入库时间", readableTime(doc.ingested_at)], ["权限组", aclLabel(doc.acl)],
    ].map(([label, value]) => `<div><small>${label}</small><strong>${escapeHtml(value)}</strong></div>`).join("");
    $("#document-integrity").textContent = data.integrity.ok ? `完整性校验通过：已入库 ${data.integrity.chunk_total} 个分块，序号从 0 连续至 ${Math.max(0, data.integrity.chunk_total - 1)}。` : "完整性校验提示：分块序号存在缺失，请重新入库检查。";
    $("#document-integrity").classList.toggle("warning", !data.integrity.ok);
    $("#document-detail-query").value = data.query || "";
    $("#document-detail-full").textContent = fullText ? "返回分块阅读" : "查看全文";
    $("#document-detail-summary").textContent = fullText ? `全文阅读模式 · 已按原始序号合并 ${data.integrity.chunk_total} 个分块。` : (data.query ? `精确匹配 “${data.query}” 的分块共 ${data.matched_total} 个。` : `共 ${data.integrity.chunk_total} 个已入库分块，按原始分块序号连续展示。`);
    $("#document-detail-chunks").innerHTML = fullText ? `<article class="document-chunk document-full-text"><header><strong>原始全文</strong><span>${data.integrity.character_total} 字符 · ${data.integrity.chunk_total} 个分块</span></header><pre>${escapeHtml(data.full_text || "暂无原始全文")}</pre></article>` : (data.chunks.length ? data.chunks.map((chunk) => `<article class="document-chunk"><header><strong>分块 ${chunk.index + 1}${chunk.title ? ` · ${escapeHtml(chunk.title)}` : ""}</strong><span>原始序号 ${chunk.index} · ${chunk.character_count} 字符</span></header><pre>${escapeHtml(chunk.content)}</pre></article>`).join("") : '<div class="table-empty">未找到包含该关键词的分块</div>');
    $(".document-pagination").hidden = fullText;
    if (!fullText) {
      $("#document-page-label").textContent = `第 ${data.page} / ${data.total_pages} 页 · ${data.matched_total} 个分块`;
      $("#document-page-jump").value = data.page;
      $("#document-page-jump").max = data.total_pages;
      $("#document-page-prev").disabled = data.page <= 1;
      $("#document-page-next").disabled = data.page >= data.total_pages;
    }
  } catch (error) {
    if (state.companyCode !== companyCode || state.documentDetail.id !== documentId) return;
    $("#document-detail-chunks").innerHTML = `<div class="table-empty">${escapeHtml(error.message)}</div>`;
  }
}

async function deleteDocument(documentId) {
  if (!documentId || !confirm("删除后该文档不会再参与检索，确定继续吗？")) return;
  try {
    await api(`/api/documents/${encodeURIComponent(documentId)}?company_code=${encodeURIComponent(state.companyCode)}`, { method: "DELETE" });
    showToast("文档及其检索索引已删除");
    await Promise.all([loadDocuments(), loadOverview()]);
  } catch (error) { showToast(error.message); }
}

$("#document-filter").addEventListener("input", renderDocuments);
$("#document-detail-close").addEventListener("click", () => $("#document-detail-dialog").close());
$("#document-detail-dialog").addEventListener("click", (event) => { if (event.target === event.currentTarget) event.currentTarget.close(); });
$("#document-detail-search").addEventListener("click", () => openDocumentDetail(state.documentDetail.id, 1, $("#document-detail-query").value.trim()));
$("#document-detail-query").addEventListener("keydown", (event) => { if (event.key === "Enter") { event.preventDefault(); $("#document-detail-search").click(); } });
$("#document-detail-full").addEventListener("click", () => openDocumentDetail(state.documentDetail.id, 1, "", !state.documentDetail.fullText));
$("#document-page-prev").addEventListener("click", () => openDocumentDetail(state.documentDetail.id, Math.max(1, state.documentDetail.page - 1)));
$("#document-page-next").addEventListener("click", () => openDocumentDetail(state.documentDetail.id, state.documentDetail.page + 1));
$("#document-page-go").addEventListener("click", () => openDocumentDetail(state.documentDetail.id, Number($("#document-page-jump").value) || 1));

function updateIngestionControls() {
  $("#file-input").accept = ".txt,.md,.pdf,.docx";
  $("#file-hint").textContent = state.ingestionDomain ? `${currentCompany()?.name} / ${state.ingestionDomain} · 支持 PDF、DOCX、MD、TXT` : "请先选择完整三级目录 · 支持 PDF、DOCX、MD、TXT";
  $("#selected-files").innerHTML = state.files.map((file) => `<span class="file-chip">${escapeHtml(file.name)} · ${(file.size / 1024).toFixed(1)} KB</span>`).join("");
  const ready = Boolean(CatalogControls.model(currentCompany()?.leaves || [], state.ingestionSelection).leaf);
  $("#ingestion-submit").disabled = state.ingestionBusy || !state.files.length || !ready;
  $("#file-input").disabled = state.ingestionBusy || !ready;
  $("#drop-zone").classList.toggle("disabled", !ready || state.ingestionBusy);
}

$("#ingestion-company").addEventListener("change", (event) => switchCompany(event.target.value));
$$('#ingestion-mode button').forEach((button) => button.addEventListener("click", () => { state.ingestionMode = button.dataset.mode; $$('#ingestion-mode button').forEach((item) => item.classList.toggle("active", item === button)); }));
$("#file-input").addEventListener("change", (event) => { state.files = [...event.target.files]; updateIngestionControls(); });
["dragenter", "dragover"].forEach((name) => $("#drop-zone").addEventListener(name, (event) => { event.preventDefault(); $("#drop-zone").classList.add("dragover"); }));
["dragleave", "drop"].forEach((name) => $("#drop-zone").addEventListener(name, (event) => { event.preventDefault(); $("#drop-zone").classList.remove("dragover"); }));
$("#drop-zone").addEventListener("drop", (event) => { if (!state.ingestionDomain || state.ingestionBusy) return; state.files = [...event.dataTransfer.files]; updateIngestionControls(); });

function renderIngestionStatus(job) {
  const status = $("#ingestion-status");
  const progress = Math.max(0, Math.min(100, Number(job.progress) || 0));
  const labels = { uploading: "正在上传", queued: "等待入库", processing: "正在入库", succeeded: "入库完成", failed: "入库失败" };
  status.hidden = false;
  status.dataset.status = job.status;
  status.innerHTML = `<div class="ingestion-progress-head"><strong>${labels[job.status] || "正在处理"}</strong><span>${progress}%</span></div><progress max="100" value="${progress}" aria-label="入库进度"></progress><p class="ingestion-progress-message">${escapeHtml(job.message || "")}</p>`;
}

async function pollIngestion(jobId) {
  for (let attempt = 0; attempt < 360; attempt += 1) {
    const job = await api(`/api/ingestion/jobs/${jobId}`);
    renderIngestionStatus(job);
    if (["succeeded", "failed"].includes(job.status)) {
      if (job.status === "succeeded") { state.files = []; $("#file-input").value = ""; updateIngestionControls(); loadOverview(); }
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 2000));
  }
}

$("#ingestion-submit").addEventListener("click", async () => {
  if (!state.files.length || !state.ingestionDomain || state.ingestionBusy) return;
  state.ingestionBusy = true;
  renderCatalogCascade("ingestion");
  $("#ingestion-company").disabled = true;
  updateIngestionControls();
  const form = new FormData();
  form.append("company_code", state.companyCode);
  form.append("domain_path", state.ingestionDomain);
  form.append("mode", state.ingestionMode);
  state.files.forEach((file) => form.append("files", file));
  $("#ingestion-submit").disabled = true;
  const status = $("#ingestion-status"); renderIngestionStatus({ status: "uploading", progress: 0, message: "正在安全上传文件…" });
  try { const job = await api("/api/ingestion/upload", { method: "POST", body: form }); showToast("文件已上传，入库任务开始执行"); await pollIngestion(job.id); }
  catch (error) { renderIngestionStatus({ status: "failed", progress: $("progress", status)?.value || 0, message: `上传或入库失败：${error.message}` }); }
  finally { state.ingestionBusy = false; $("#ingestion-company").disabled = false; renderCatalogCascade("ingestion"); updateIngestionControls(); }
});

async function loadUsers() {
  try {
    const [data, credentialData] = await Promise.all([api("/api/users"), api("/api/credentials")]);
    const users = data.users || [];
    const credentials = credentialData.clients || [];
    state.users = users;
    state.credentials = credentials;
    const credentialNames = new Map(credentials.map((item) => [item.id, item.name]));
    $("#user-total").textContent = users.length;
    $("#user-admins").textContent = users.filter((item) => item.role !== "user").length;
    $("#user-normal").textContent = users.filter((item) => item.role === "user").length;
    $("#user-list").innerHTML = users.length ? users.map((user) => {
      const bindings = (user.credential_ids || []).map((id) => credentialNames.get(id)).filter(Boolean);
      const bindingText = bindings.length ? bindings.join("、") : "未绑定";
      const keyText = user.api_key_prefix ? `${user.api_key_prefix}…` : "未生成";
      const actions = user.role === "superadmin"
        ? '<span class="table-action">唯一超级管理员</span>'
        : `<div class="table-actions"><button class="table-action" data-user-edit="${escapeHtml(user.id)}">编辑</button><button class="table-action" data-user-key="${escapeHtml(user.id)}">重置密钥</button><button class="table-action danger" data-user-delete="${escapeHtml(user.id)}">删除</button></div>`;
      return `<tr><td><strong>${escapeHtml(user.name)}</strong><small>${escapeHtml(user.position)}</small></td><td>${escapeHtml(user.username)}</td><td>${escapeHtml(user.role_label || user.role)}</td><td><span class="credential-scope ${bindings.length ? "" : "muted"}">${escapeHtml(bindingText)}</span></td><td><code class="key-prefix">${escapeHtml(keyText)}</code></td><td>${actions}</td></tr>`;
    }).join("") : '<tr><td colspan="6" class="table-empty">暂无用户，可创建第一个管理账号</td></tr>';
    $$('[data-user-delete]').forEach((button) => button.addEventListener("click", () => deleteUser(button.dataset.userDelete)));
    $$('[data-user-edit]').forEach((button) => button.addEventListener("click", () => openUserDialog(state.users.find((item) => item.id === button.dataset.userEdit))));
    $$('[data-user-key]').forEach((button) => button.addEventListener("click", () => rotateUserKey(button.dataset.userKey)));
  } catch (error) { $("#user-list").innerHTML = `<tr><td colspan="6" class="table-empty">${escapeHtml(error.message)}</td></tr>`; }
}

async function deleteUser(id) {
  if (!confirm("确定删除这个用户吗？")) return;
  try { await api(`/api/users/${id}`, { method: "DELETE" }); showToast("用户已删除"); loadUsers(); } catch (error) { showToast(error.message); }
}

function renderUserCredentialOptions(selectedIds = []) {
  const selected = new Set(selectedIds);
  const target = $("#user-credential-options");
  target.innerHTML = state.credentials.length ? state.credentials.map((credential) => {
    const scopes = credential.knowledge_scopes || [];
    const leafCount = scopes.reduce((count, scope) => count + (scope.domain_paths || []).length, 0);
    const capability = credential.can_answer ? "检索 + 回答" : "仅证据检索";
    return `<label class="user-credential-option"><input type="checkbox" name="credential_ids" value="${escapeHtml(credential.id)}" ${selected.has(credential.id) ? "checked" : ""}><span><b>${escapeHtml(credential.name)}</b><small>${escapeHtml(capability)} · ${leafCount || "旧版"} 个三级事项 · ${credential.enabled ? "启用" : "停用"}</small></span></label>`;
  }).join("") : '<p class="credential-empty">暂无可绑定凭证，请先在智能体配置中创建。</p>';
  $$('input[name="credential_ids"]', target).forEach((input) => input.addEventListener("change", updateUserCredentialSummary));
  updateUserCredentialSummary();
}

function updateUserCredentialSummary() {
  const count = $$('input[name="credential_ids"]:checked', $("#user-form")).length;
  $("#user-credential-summary").textContent = `已选择 ${count} 个凭证`;
}

async function openUserDialog(user = null) {
  if (!state.credentials.length) {
    try { state.credentials = (await api("/api/credentials")).clients || []; } catch (error) { showToast(error.message); return; }
  }
  const form = $("#user-form");
  form.reset();
  form.elements.id.value = user?.id || "";
  form.elements.name.value = user?.name || "";
  form.elements.position.value = user?.position || "";
  form.elements.username.value = user?.username || "";
  form.elements.role.value = user?.role === "admin" ? "admin" : "user";
  form.elements.password.required = !user;
  form.elements.password.value = "";
  $("#user-password-label").textContent = user ? "新密码（留空不修改）" : "初始密码";
  $("#user-dialog-title").textContent = user ? "编辑用户" : "新建用户";
  $("#user-submit").textContent = user ? "保存修改" : "创建用户";
  renderUserCredentialOptions(user?.credential_ids || []);
  $("#user-dialog").showModal();
}

async function rotateUserKey(id) {
  const user = state.users.find((item) => item.id === id);
  if (!confirm(`确定重置 ${user?.name || "该用户"} 的接口密钥吗？旧密钥将立即失效。`)) return;
  try {
    const result = await api(`/api/users/${id}/api-key`, { method: "POST" });
    showCreatedKey(result.api_key, "用户接口密钥已重置", "外部系统使用此密钥调用接口；权限来自该用户绑定的多个凭证。旧密钥已失效。", true);
    loadUsers();
  } catch (error) { showToast(error.message); }
}

function showCreatedKey(key, title = "请立即保存 API Key", note = "出于安全原因，密钥仅显示这一次。", userKey = false) {
  $("#key-dialog-title").textContent = title;
  $("#key-dialog-note").textContent = note;
  $("#created-key").textContent = key;
  $("#key-dialog").dataset.userKey = userKey ? "true" : "false";
  $("#key-dialog").showModal();
}

$("#user-add").addEventListener("click", () => openUserDialog());
$("#user-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const id = form.get("id");
  const payload = { name: form.get("name"), position: form.get("position"), username: form.get("username"), password: form.get("password"), role: form.get("role"), credential_ids: form.getAll("credential_ids") };
  try {
    const user = await api(id ? `/api/users/${id}` : "/api/users", { method: id ? "PUT" : "POST", body: JSON.stringify(payload) });
    $("#user-dialog").close(); event.currentTarget.reset();
    if (user.api_key) showCreatedKey(user.api_key, "用户接口密钥已创建", "外部系统使用此密钥调用接口；权限为所选凭证的有效权限并集。密钥仅显示这一次。", true);
    else showToast("用户已更新");
    loadUsers();
  } catch (error) { showToast(error.message); }
});

async function loadAgents() {
  try {
    const data = await api("/api/credentials");
    const clients = data.clients || [];
    $("#stat-agents").textContent = clients.length;
    $("#agent-list").innerHTML = clients.length ? clients.map((client) => {
      const scopes = client.knowledge_scopes || [];
      const leafCount = scopes.reduce((count, scope) => count + (scope.domain_paths || []).length, 0);
      const scopeText = scopes.length
        ? `${scopes.map((scope) => state.companies.find((item) => item.code === scope.company_code)?.name || scope.company_code).join("、")} · ${leafCount} 个三级事项`
        : (client.domains || []).join("、") || "旧版全域权限";
      return `<tr><td><strong>${escapeHtml(client.name)}</strong><small>${escapeHtml(client.id)}</small></td><td><span class="credential-scope">${escapeHtml(scopeText)}</span></td><td>${client.can_answer ? "检索 + 回答" : "仅证据检索"}</td><td>${client.rate_limit_per_min || 60} 次/分钟</td><td><span class="status-pill">${client.enabled ? "启用" : "停用"}</span></td><td><button class="table-action danger" data-agent-delete="${escapeHtml(client.id)}">删除</button></td></tr>`;
    }).join("") : '<tr><td colspan="6" class="table-empty">暂无智能体凭证</td></tr>';
    $$('[data-agent-delete]').forEach((button) => button.addEventListener("click", () => deleteAgent(button.dataset.agentDelete)));
  } catch (error) { $("#agent-list").innerHTML = `<tr><td colspan="6" class="table-empty">${escapeHtml(error.message)}</td></tr>`; }
}

async function deleteAgent(id) {
  if (!confirm("删除后原 API Key 将立即失效，确定继续吗？")) return;
  try { await api(`/api/credentials/${id}`, { method: "DELETE" }); showToast("凭证已删除"); loadAgents(); } catch (error) { showToast(error.message); }
}

function agentScopeKey(companyCode, domainPath) { return `${companyCode}::${domainPath}`; }

function agentLeaves(companyCode, prefix = "") {
  const company = state.companies.find((item) => item.code === companyCode);
  return (company?.leaves || []).filter((leaf) => !prefix || leaf.domain_path === prefix || leaf.domain_path.startsWith(`${prefix}/`));
}

function renderAgentScopeTree() {
  $("#agent-scope-tree").innerHTML = state.companies.map((company, companyIndex) => {
    const roots = [...new Set(company.leaves.map((leaf) => leaf.level1))];
    const rootMarkup = roots.map((root) => {
      const rootLeaves = company.leaves.filter((leaf) => leaf.level1 === root);
      const secondaryNames = [...new Set(rootLeaves.map((leaf) => leaf.level2))];
      return `<details class="agent-scope-level" ${companyIndex === 0 ? "open" : ""}><summary><label><input type="checkbox" data-agent-prefix="${escapeHtml(root)}" data-agent-company="${escapeHtml(company.code)}"><span>${escapeHtml(root)}</span></label><small>${rootLeaves.length} 个三级事项</small><i>⌄</i></summary><div class="agent-scope-secondary">${secondaryNames.map((secondary) => {
        const prefix = `${root}/${secondary}`;
        const leaves = rootLeaves.filter((leaf) => leaf.level2 === secondary);
        return `<section><header><label><input type="checkbox" data-agent-prefix="${escapeHtml(prefix)}" data-agent-company="${escapeHtml(company.code)}"><b>${escapeHtml(secondary)}</b></label><small>${leaves.length} 项</small></header><div class="agent-scope-leaves">${leaves.map((leaf) => `<label title="${escapeHtml(leaf.classification_standard || leaf.domain_path)}"><input type="checkbox" data-agent-leaf="${escapeHtml(leaf.domain_path)}" data-agent-company="${escapeHtml(company.code)}"><span>${escapeHtml(leaf.level3)}</span></label>`).join("")}</div></section>`;
      }).join("")}</div></details>`;
    }).join("");
    return `<details class="agent-scope-platform" ${companyIndex === 0 ? "open" : ""}><summary><label><input type="checkbox" data-agent-platform="${escapeHtml(company.code)}" data-agent-company="${escapeHtml(company.code)}"><span><b>${escapeHtml(company.name)}</b><small>${company.leaves.length} 个三级事项</small></span></label><i>⌄</i></summary><div class="agent-scope-levels">${rootMarkup}</div></details>`;
  }).join("");
  updateAgentScopeStates();
}

function updateAgentScopeStates() {
  $$("[data-agent-leaf]", $("#agent-scope-tree")).forEach((input) => {
    input.checked = state.agentScopePaths.has(agentScopeKey(input.dataset.agentCompany, input.dataset.agentLeaf));
  });
  $$("[data-agent-prefix]", $("#agent-scope-tree")).forEach((input) => {
    const leaves = agentLeaves(input.dataset.agentCompany, input.dataset.agentPrefix);
    const selected = leaves.filter((leaf) => state.agentScopePaths.has(agentScopeKey(input.dataset.agentCompany, leaf.domain_path))).length;
    input.checked = selected === leaves.length && leaves.length > 0;
    input.indeterminate = selected > 0 && selected < leaves.length;
  });
  $$("[data-agent-platform]", $("#agent-scope-tree")).forEach((input) => {
    const leaves = agentLeaves(input.dataset.agentCompany);
    const selected = leaves.filter((leaf) => state.agentScopePaths.has(agentScopeKey(input.dataset.agentCompany, leaf.domain_path))).length;
    input.checked = selected === leaves.length && leaves.length > 0;
    input.indeterminate = selected > 0 && selected < leaves.length;
  });
  const count = state.agentScopePaths.size;
  const platforms = state.companies.filter((company) => company.leaves.some((leaf) => state.agentScopePaths.has(agentScopeKey(company.code, leaf.domain_path)))).length;
  $("#agent-scope-summary").textContent = count ? `已选 ${platforms} 个平台 · ${count} 个三级事项` : "已选择 0 个三级目录";
  $("#agent-create-submit").disabled = count === 0;
}

$("#agent-scope-tree").addEventListener("click", (event) => {
  if (event.target.matches('input[type="checkbox"]')) event.stopPropagation();
});
$("#agent-scope-tree").addEventListener("change", (event) => {
  const input = event.target.closest('input[type="checkbox"]');
  if (!input) return;
  const code = input.dataset.agentCompany;
  const leaves = input.dataset.agentLeaf
    ? agentLeaves(code, input.dataset.agentLeaf)
    : agentLeaves(code, input.dataset.agentPrefix || "");
  leaves.forEach((leaf) => {
    const key = agentScopeKey(code, leaf.domain_path);
    if (input.checked) state.agentScopePaths.add(key); else state.agentScopePaths.delete(key);
  });
  updateAgentScopeStates();
});

$("#agent-add").addEventListener("click", () => {
  $("#agent-form").reset();
  state.agentScopePaths = new Set();
  renderAgentScopeTree();
  $("#agent-dialog").showModal();
});
$("#agent-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const formElement = event.currentTarget;
  const form = new FormData(formElement);
  try {
    const knowledgeScopes = state.companies.map((company) => ({
      company_code: company.code,
      domain_paths: company.leaves.filter((leaf) => state.agentScopePaths.has(agentScopeKey(company.code, leaf.domain_path))).map((leaf) => leaf.domain_path),
    })).filter((scope) => scope.domain_paths.length);
    const client = await api("/api/credentials", { method: "POST", body: JSON.stringify({ name: form.get("name"), knowledge_scopes: knowledgeScopes, can_answer: form.get("answer") === "on", rate_limit_per_min: Number(form.get("rate") || 60) }) });
    $("#agent-dialog").close(); formElement.reset(); state.agentScopePaths = new Set(); showCreatedKey(client.api_key); loadAgents();
  } catch (error) { showToast(error.message); }
});

$("#copy-key").addEventListener("click", async () => { await navigator.clipboard.writeText($("#created-key").textContent); showToast("API Key 已复制"); });
$$('[data-dialog-close]').forEach((button) => button.addEventListener("click", () => button.closest("dialog").close()));

const modelLabels = { llm: ["LLM", "回答与语义分析模型"], embedding: ["EMB", "向量化模型"], rerank: ["RANK", "相关性重排模型"] };
async function loadModels() {
  try {
    const data = await api("/api/models");
    $("#models-list").innerHTML = Object.entries(modelLabels).map(([kind, label]) => { const config = data[kind] || {}; return `<article class="surface model-card" data-kind="${kind}"><div class="model-card-head"><span class="model-symbol">${label[0]}</span><div><h3>${label[1]}</h3><p>${config.model ? "已配置" : "等待配置"}</p></div></div><div class="model-form"><label>API 地址<input data-key="base_url" value="${escapeHtml(config.base_url || "")}" placeholder="https://api.example.com/v1"></label><label>模型名称<input data-key="model" value="${escapeHtml(config.model || "")}" placeholder="模型标识"></label><label>API Key<input data-key="api_key" type="password" value="" placeholder="${escapeHtml(config.api_key || "留空保持不变")}"></label><div class="model-actions"><button class="button secondary" type="button" data-model-test="${kind}">测试连通性</button><button class="button primary" type="button" data-model-save="${kind}">保存配置</button></div><p class="model-test-result" aria-live="polite"></p></div></article>`; }).join("");
    $$('[data-model-save]').forEach((button) => button.addEventListener("click", () => saveModel(button.dataset.modelSave, button.closest(".model-card"))));
    $$('[data-model-test]').forEach((button) => button.addEventListener("click", () => testModel(button.dataset.modelTest, button.closest(".model-card"), button)));
  } catch (error) { $("#models-list").innerHTML = `<div class="table-empty">${escapeHtml(error.message)}</div>`; }
}

async function saveModel(kind, card) {
  const body = {}; $$('[data-key]', card).forEach((input) => { if (input.value.trim()) body[input.dataset.key] = input.value.trim(); });
  try { await api(`/api/models/${kind}`, { method: "PUT", body: JSON.stringify(body) }); showToast("模型配置已保存"); loadModels(); } catch (error) { showToast(error.message); }
}

async function testModel(kind, card, button) {
  const values = {};
  $$('[data-key]', card).forEach((input) => { if (input.value.trim()) values[input.dataset.key] = input.value.trim(); });
  const result = $(".model-test-result", card);
  button.disabled = true;
  button.textContent = "测试中…";
  result.className = "model-test-result";
  result.textContent = "正在发起真实模型请求…";
  try {
    const data = await api(`/api/models/${kind}/test`, { method: "POST", body: JSON.stringify(values) });
    result.classList.add("success");
    result.textContent = `${data.message} · ${data.latency_ms} ms`;
  } catch (error) {
    result.classList.add("error");
    result.textContent = error.message;
  } finally {
    button.disabled = false;
    button.textContent = "测试连通性";
  }
}

function renderCompanyControls() {
  const options = state.companies.map((company) => `<option value="${escapeHtml(company.code)}">${escapeHtml(company.name)} · ${escapeHtml(company.type)}</option>`).join("");
  $("#ingestion-company").innerHTML = options;
  $("#ingestion-company").value = state.companyCode;
  renderCompanySwitcher();
  state.ingestionSelection = ["", "", ""];
  state.searchSelection = ["", "", ""];
  state.documentSelection = ["", "", ""];
  state.ingestionDomain = "";
  state.documentDomain = "all";
  ["ingestion", "search", "document"].forEach(renderCatalogCascade);
  $("#file-input").value = "";
  $("#stat-platform-label").textContent = `${currentCompany()?.name || "当前平台"}业务资料`;
  updateIngestionControls();
}

function renderCatalogCascade(context) {
  const selection = state[`${context}Selection`];
  const leaves = currentCompany()?.leaves || [];
  const model = CatalogControls.model(leaves, selection);
  const target = context === "ingestion" ? $("#ingestion-domain") : context === "search" ? $("#search-domains") : $("#document-domain");
  if (context === "ingestion") {
    renderIngestionTree(target, leaves, selection, model);
    return;
  }
  target.innerHTML = model.options.map((options, depth) => `<label><span>${["一级目录", "二级目录", "三级目录"][depth]}</span><select data-catalog-context="${context}" data-depth="${depth}" ${model.disabled[depth] || (context === "ingestion" && state.ingestionBusy) ? "disabled" : ""}><option value="">${context === "ingestion" ? "请选择" : "全部"}</option>${options.map((path) => `<option value="${escapeHtml(path)}" ${selection[depth] === path ? "selected" : ""}>${escapeHtml(path.split("/").at(-1))}</option>`).join("")}</select></label>`).join("");
}

function renderIngestionTree(target, leaves, selection, model) {
  const roots = model.options[0];
  const rootLeaves = leaves.filter((leaf) => leaf.level1 === selection[0]);
  const secondaryNames = [...new Set(rootLeaves.map((leaf) => leaf.level2))];
  const secondaryLeaves = rootLeaves.filter((leaf) => `${leaf.level1}/${leaf.level2}` === selection[1]);
  const rootSelect = `<label class="catalog-root-select"><span>一级目录</span><select data-catalog-context="ingestion" data-depth="0" ${state.ingestionBusy ? "disabled" : ""}><option value="">请选择一级目录</option>${roots.map((root) => `<option value="${escapeHtml(root)}" ${selection[0] === root ? "selected" : ""}>${escapeHtml(root)}</option>`).join("")}</select></label>`;
  const tree = selection[0] ? `<section class="directory-tree" aria-label="二级目录树"><div class="tree-root"><span>一级目录</span><strong>${escapeHtml(selection[0])}</strong><small>${secondaryNames.length} 个二级目录</small></div><div class="tree-trunk" aria-hidden="true"></div><div class="tree-branches">${secondaryNames.map((name) => {
    const path = `${selection[0]}/${name}`;
    const children = rootLeaves.filter((leaf) => leaf.level2 === name);
    const files = children.reduce((count, leaf) => count + leaf.related_files.length, 0);
    return `<button type="button" class="tree-branch${selection[1] === path ? " active" : ""}" data-secondary-path="${escapeHtml(path)}" ${state.ingestionBusy ? "disabled" : ""}><span class="tree-node-dot"></span><span><b>${escapeHtml(name)}</b><small>${children.length} 个三级事项 · ${files} 项相关文件</small></span><i>›</i></button>`;
  }).join("")}</div></section>` : `<div class="directory-guide"><b>先选择一级目录</b><span>系统将展示该目录下的二级子树</span></div>`;
  const tertiary = selection[1] ? `<section class="tertiary-list"><header><div><span>三级目录</span><h4>${escapeHtml(selection[1].split("/").at(-1))}</h4></div><small>${secondaryLeaves.length} 个事项</small></header><p class="tertiary-definition">${escapeHtml(secondaryLeaves[0]?.level2_definition || "Excel 未提供释义")}</p><div class="tertiary-items">${secondaryLeaves.map((leaf) => `<button type="button" class="tertiary-item${selection[2] === leaf.domain_path ? " active" : ""}" data-leaf-path="${escapeHtml(leaf.domain_path)}" ${state.ingestionBusy ? "disabled" : ""}><span><b>${escapeHtml(leaf.level3)}</b><small>${leaf.related_files.length} 项相关文件${leaf.related_files.length ? ` · ${escapeHtml(leaf.related_files.slice(0, 2).join("、"))}${leaf.related_files.length > 2 ? "…" : ""}` : ""}</small></span><i>${selection[2] === leaf.domain_path ? "已选择" : "选择"}</i></button>`).join("")}</div></section>` : "";
  target.innerHTML = rootSelect + tree + tertiary;
  const info = $("#catalog-overview");
  info.hidden = true;
  info.innerHTML = "";
  const detail = $("#catalog-leaf-detail");
  detail.hidden = !model.leaf;
  detail.innerHTML = model.leaf ? `<h3>${escapeHtml(model.leaf.level3)}</h3><p class="catalog-path">${escapeHtml(currentCompany().name)} / ${escapeHtml(model.leaf.domain_path.replaceAll("/", " / "))}</p><dl><dt>归类标准</dt><dd>${escapeHtml(model.leaf.classification_standard || "Excel 未指定")}</dd></dl><h4>相关文件完整清单 · ${model.leaf.related_files.length} 项</h4><ol>${model.leaf.file_items.map((item) => `<li>${escapeHtml(item.name)}${item.classification_standard && model.leaf.classification_standards.length > 1 ? `<small>归类：${escapeHtml(item.classification_standard)}</small>` : ""}</li>`).join("")}</ol>` : "";
}

function resetIngestionFiles() {
  state.ingestionDomain = state.ingestionSelection[2];
  state.files = [];
  $("#file-input").value = "";
  updateIngestionControls();
}

$("#ingestion-domain").addEventListener("click", (event) => {
  const secondary = event.target.closest("[data-secondary-path]");
  const leaf = event.target.closest("[data-leaf-path]");
  if (secondary) state.ingestionSelection = CatalogControls.change(state.ingestionSelection, 1, secondary.dataset.secondaryPath);
  if (leaf) state.ingestionSelection = CatalogControls.change(state.ingestionSelection, 2, leaf.dataset.leafPath);
  if (!secondary && !leaf) return;
  renderCatalogCascade("ingestion");
  resetIngestionFiles();
});

document.addEventListener("change", (event) => {
  const select = event.target.closest("select[data-catalog-context]");
  if (!select) return;
  const context = select.dataset.catalogContext;
  state[`${context}Selection`] = CatalogControls.change(state[`${context}Selection`], Number(select.dataset.depth), select.value);
  renderCatalogCascade(context);
  if (context === "ingestion") {
    resetIngestionFiles();
  }
  if (context === "document") { state.documentDomain = CatalogControls.prefix(state.documentSelection) || "all"; loadDocuments(); }
  if (context === "search") { $("#result-summary").textContent = "检索范围已更新"; $("#search-results").innerHTML = ""; }
});

function renderCompanySwitcher() {
  const current = currentCompany();
  $("#company-select .company-trigger-main").textContent = current ? `${current.name} · 平台` : "选择平台";
  $("#company-menu").innerHTML = state.companies.map((company) => {
    const selected = company.code === state.companyCode;
    return `<button class="company-option${selected ? " active" : ""}" type="button" role="option" aria-selected="${selected}" data-company-code="${escapeHtml(company.code)}"><span class="company-option-mark">${escapeHtml(company.name.slice(0, 1))}</span><span class="company-option-copy"><b>${escapeHtml(company.name)}</b><small>${escapeHtml(company.type)}</small></span>${selected ? '<span class="company-option-check" aria-hidden="true">✓</span>' : ""}</button>`;
  }).join("");
}

function closeCompanySwitcher() {
  $("#company-switcher").classList.remove("open");
  $("#company-select").setAttribute("aria-expanded", "false");
}

async function switchCompany(companyCode) {
  if (state.ingestionBusy) return showToast("正在入库，请等待当前任务完成后切换平台");
  if (!state.companies.some((item) => item.code === companyCode)) return;
  state.companyCode = companyCode;
  state.files = [];
  state.documents = [];
  $("#document-detail-dialog").close();
  state.documentDetail = { id: "", page: 1, query: "", fullText: false };
  renderCompanyControls();
  if (state.view === "overview") await loadOverview();
  if (state.view === "documents") await loadDocuments();
  if (state.view === "search") { $("#search-results").className = "search-results empty-state"; $("#search-results").innerHTML = '<div class="empty-illustration">⌕</div><h3>已切换平台</h3><p>检索仅返回当前平台的资料。</p>'; }
}

async function bootstrapCompanies() {
  try {
    const data = await api("/api/companies");
    state.companies = data.companies || [];
    if (!state.companies.some((item) => item.code === state.companyCode)) state.companyCode = state.companies[0]?.code || "";
    renderCompanyControls();
  } catch (error) { showToast(`平台目录加载失败：${error.message}`); }
}

$("#company-select").addEventListener("click", () => {
  const switcher = $("#company-switcher");
  const open = switcher.classList.toggle("open");
  $("#company-select").setAttribute("aria-expanded", String(open));
});
$("#company-menu").addEventListener("click", async (event) => {
  const option = event.target.closest("[data-company-code]");
  if (!option) return;
  closeCompanySwitcher();
  await switchCompany(option.dataset.companyCode);
});
document.addEventListener("click", (event) => {
  if (!$("#company-switcher").contains(event.target)) closeCompanySwitcher();
});
window.addEventListener("keydown", (event) => { if (event.key === "Escape") closeCompanySwitcher(); });

bootstrapCompanies().then(() => setView(location.hash.slice(1) || "overview", false));
