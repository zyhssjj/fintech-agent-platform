import React from "react";
import { Bot, FileText, FileUp, Library, Mail, MessageSquare, Plus, Send, X } from "lucide-react";
import {
  communicationApi,
  type DailySummary,
  type DocumentItem,
  type EmailItem,
  type LPRecipient,
  type MailboxAccount,
  type MailboxPayload,
  type PersonRef,
  type TemplateAssistResponse,
  type TemplateItem,
  type TemplateVariable,
  type TemplateVersion,
} from "@/module-agent/api/communication";
import { useAuthStore } from "@/base/stores/authStore";

type TabKey = "overview" | "mail" | "templates" | "documents";

type DocumentFormState = {
  sender_mailbox_id: number;
  lp_user_id: string;
  recipient_name: string;
  recipient_email: string;
  title: string;
  fund_name: string;
  lp_name: string;
  notice_date: string;
};

type TemplateDraftFormState = {
  name: string;
  category: string;
  scene: string;
  usage_boundary: string;
  goal: string;
};

type TemplateChatMessage = {
  id: string;
  role: "user" | "assistant";
  text: string;
};

type TemplateFieldPlacement = "subject" | "html" | "both";

type TemplateFieldPlacementState = {
  variableName: string;
  variableLabel: string;
  attempt: number;
};

const fixedDocumentVariableFields: Record<string, keyof DocumentFormState> = {
  fund_name: "fund_name",
  lp_name: "lp_name",
  notice_date: "notice_date",
  recipient_name: "recipient_name",
  recipient_email: "recipient_email",
  title: "title",
  document_title: "title",
};

function toRecipientRefs(emails: string[]): PersonRef[] {
  const seen = new Set<string>();
  return emails
    .map((item) => item.trim())
    .filter((email) => {
      const key = email.toLowerCase();
      if (!key || seen.has(key)) return false;
      seen.add(key);
      return true;
    })
    .map((email) => ({ name: email, email }));
}

function deriveDocumentVariableValue(
  variableName: string,
  form: DocumentFormState,
  extraValues: Record<string, string>,
) {
  if (fixedDocumentVariableFields[variableName]) {
    return String(form[fixedDocumentVariableFields[variableName]] ?? "");
  }
  return extraValues[variableName] || "";
}

function getVariableLabel(variable: TemplateVariable) {
  return variable.variable_label || variable.variable_name;
}

function templateStatusLabel(status: string) {
  const map: Record<string, string> = {
    draft: "草稿",
    pending_approval: "待审批",
    approved: "已审批",
    published: "已发布",
    deleted: "已删除",
  };
  return map[status] || status;
}

function templateSourceLabel(source?: string) {
  const map: Record<string, string> = {
    system_builtin: "系统内置",
    user_private: "我的私人模板",
    published_public: "已发布公共模板",
  };
  return source ? map[source] || source : "-";
}

function templateCategoryLabel(category?: string) {
  const map: Record<string, string> = {
    notice: "通知",
    email: "邮件",
    cooperation: "合作函件",
  };
  return category ? map[category] || category : "未分类";
}

function documentStatusLabel(status: string) {
  const map: Record<string, string> = {
    draft: "草稿",
    rendered: "已预览",
    pending_level_1: "待一级审批",
    pending_level_2: "待二级审批",
    pending_approval: "待审批",
    approved: "审批通过",
    sent: "已发送",
    rejected: "已驳回",
  };
  return map[status] || status;
}

function sendResultLabel(value?: string) {
  const map: Record<string, string> = {
    sent: "已发送至收件人邮箱",
    delivered: "投递成功",
    failed: "发送失败",
  };
  return value ? map[value] || value : "-";
}

function canSubmitDocument(status: string) {
  return status === "draft" || status === "rendered";
}

function canSendDocument(status: string, hasFormalDocx: boolean, initiatorUserID: string, currentUserID?: string) {
  return hasFormalDocx && status === "approved" && Boolean(currentUserID) && initiatorUserID === currentUserID;
}

function canUseMailboxAsSender(mailbox: MailboxAccount) {
  return mailbox.oauth_provider !== "microsoft_graph" || mailbox.oauth_connected === true;
}

function canDownloadDocument(status: string, hasFormalDocx: boolean) {
  return hasFormalDocx && ["approved", "sent"].includes(status);
}

function escapeHtml(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function renderClientTemplate(value: string, variables: Record<string, unknown>) {
  return value.replace(/\{\{\s*([a-zA-Z0-9_]+)\s*\}\}|\{\s*([a-zA-Z0-9_]+)\s*\}/g, (_, a, b) => {
    const key = a || b;
    const raw = variables[key];
    return escapeHtml(raw == null ? "" : String(raw));
  });
}

function renderTemplatePlaceholderLabels(value: string, variables: TemplateVariable[]) {
  const labelMap = new Map(
    variables.map((item) => [item.variable_name, item.variable_label || item.variable_name] as const),
  );

  return value.replace(/\{\{\s*([a-zA-Z0-9_]+)\s*\}\}|\{\s*([a-zA-Z0-9_]+)\s*\}/g, (_, a, b) => {
    const key = a || b;
    const label = labelMap.get(key) || key;
    return `{${label}}`;
  });
}

function extractTemplateVariableNames(value?: string) {
  const matches = Array.from((value || "").matchAll(/\{\{\s*([a-zA-Z0-9_]+)\s*\}\}|\{\s*([a-zA-Z0-9_]+)\s*\}/g));
  return Array.from(new Set(matches.map((match) => match[1] || match[2]).filter(Boolean)));
}

function normalizeTemplateVariableKey(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^\w\u4e00-\u9fa5]+/g, "_")
    .replace(/^_+|_+$/g, "");
}

function inferVariableNameFromLabel(label: string) {
  const normalized = label.trim().toLowerCase();
  const map: Record<string, string> = {
    称呼语: "salutation",
    称谓: "salutation",
    收件人称呼: "salutation",
    问候语: "greeting",
    基金名称: "fund_name",
    lp名称: "lp_name",
    lp名: "lp_name",
    收件人名称: "recipient_name",
    收件对象: "recipient_name",
    通知日期: "notice_date",
    报告期间: "reporting_period",
    基金经理: "manager_name",
    合作事项: "cooperation_subject",
    合作方名称: "partner_name",
    被投企业名称: "portfolio_name",
    电话号码: "phone_number",
    联系电话: "phone_number",
  };
  return map[normalized] || normalizeTemplateVariableKey(label);
}

function inferVariableLabelFromKey(variableName: string) {
  const map: Record<string, string> = {
    fund_name: "基金名称",
    lp_name: "LP 名称",
    recipient_name: "收件人名称",
    notice_date: "通知日期",
    reporting_period: "报告期间",
    manager_name: "基金经理",
    cooperation_subject: "合作事项",
    partner_name: "合作方名称",
    portfolio_name: "被投企业名称",
    salutation: "称呼语",
    greeting: "问候语",
    phone_number: "电话号码",
  };
  return map[variableName] || variableName.replaceAll("_", " ");
}

function cleanFieldLabelCandidate(value?: string) {
  const normalized = (value || "")
    .trim()
    .replace(/^(?:字段|变量|为|叫|名为|名称为|是)\s*/g, "")
    .replace(/[，,。.;；].*$/g, "")
    .trim();
  const blocked = new Set(["字段", "变量", "新增", "添加", "增加", "加入", "补充", "一个"]);
  return blocked.has(normalized) ? "" : normalized;
}

function extractFieldIntent(message: string) {
  const normalized = message.trim().replace(/[，、：]/g, ":");
  if (!/(新增|添加|增加).{0,6}(字段|变量)/.test(normalized)) return null;

  const labelPattern = "([a-zA-Z][a-zA-Z0-9_]*|[\\u4e00-\\u9fa5A-Za-z0-9_]{2,40})";
  const patterns = [
    new RegExp(`(?:新增|添加|增加)\\s*(?:字段|变量)\\s*[,:\\s]+${labelPattern}`),
    new RegExp(`(?:字段|变量)\\s*[,:\\s]+${labelPattern}`),
    new RegExp(`(?:新增|添加|增加)\\s*${labelPattern}\\s*(?:字段|变量)`),
    new RegExp(`(?:新增|添加|增加)\\s*${labelPattern}`),
  ];
  const rawLabel = cleanFieldLabelCandidate(patterns.map((pattern) => normalized.match(pattern)?.[1]).find(Boolean));
  if (!rawLabel) return null;

  const variableName = /^[a-zA-Z][a-zA-Z0-9_]*$/.test(rawLabel) ? normalizeTemplateVariableKey(rawLabel) : inferVariableNameFromLabel(rawLabel);
  const variableLabel = /[\u4e00-\u9fa5]/.test(rawLabel) ? rawLabel : inferVariableLabelFromKey(variableName);
  return { variableName, variableLabel };
}

function inferPlacementFromMessage(message: string): TemplateFieldPlacement | null {
  const normalized = message.replace(/\s+/g, "");
  if (/(主题|标题).*(正文|html|邮件正文)/i.test(normalized) || /(正文|html|邮件正文).*(主题|标题)/i.test(normalized)) {
    return "both";
  }
  if (/(主题|标题)/.test(normalized) && !/(正文|html|邮件正文)/i.test(normalized)) {
    return "subject";
  }
  if (/(正文|html|邮件正文)/i.test(normalized) && !/(主题|标题)/.test(normalized)) {
    return "html";
  }
  return null;
}

function buildPlacementClarificationMessage(variableLabel: string, attempt: number) {
  if (attempt <= 1) {
    return `你要新增字段“${variableLabel}”，但我还不能安全判断它放在哪里。请直接回复这 3 种之一：仅主题、仅正文、主题和正文。`;
  }
  return `字段“${variableLabel}”还没有明确落位。给你 3 个建议位置，请直接回复其中一个：\n1. 仅主题：适合报告期间、批次、函件编号\n2. 仅正文：适合基金经理、合作事项、补充说明\n3. 主题和正文：适合基金名称、LP 名称、通知日期`;
}

function injectFieldIntoSubjectTemplate(subjectTemplate: string, variableName: string) {
  const placeholder = `{${variableName}}`;
  if (!subjectTemplate.trim()) return placeholder;
  if (extractTemplateVariableNames(subjectTemplate).includes(variableName)) return subjectTemplate;
  return `${subjectTemplate.trim()} | ${placeholder}`;
}

function injectFieldIntoHtmlTemplate(htmlTemplate: string, variableName: string, variableLabel: string) {
  const placeholder = `{${variableName}}`;
  if (extractTemplateVariableNames(htmlTemplate).includes(variableName)) return htmlTemplate;

  const infoLine = `${variableLabel}：${placeholder}`;
  if (!htmlTemplate.trim()) return `<p>${infoLine}</p>`;

  if (/<\/div>\s*$/i.test(htmlTemplate)) {
    return htmlTemplate.replace(/<\/div>\s*$/i, `<p>${infoLine}</p></div>`);
  }
  if (/<\/body>\s*$/i.test(htmlTemplate)) {
    return htmlTemplate.replace(/<\/body>\s*$/i, `<p>${infoLine}</p></body>`);
  }
  return `${htmlTemplate}\n<p>${infoLine}</p>`;
}

function buildClientPreviewHtml(value: string, variables: Record<string, unknown>) {
  const normalized = value.trim();
  if (!normalized) return "";
  const rendered = renderClientTemplate(normalized, variables);
  if (/<(p|div|br|ul|ol|li|table|tbody|thead|tr|td|th)\b/i.test(rendered)) {
    return rendered;
  }
  return rendered
    .split(/\n\s*\n/)
    .map((block) => block.trim())
    .filter(Boolean)
    .map((block) => `<p>${block.split("\n").map((line) => line.trim()).filter(Boolean).join("<br/>")}</p>`)
    .join("");
}

function htmlToEditableText(value: string) {
  const normalized = (value || "").trim();
  if (!normalized) return "";
  if (!/<[a-z][\s\S]*>/i.test(normalized)) return normalized;
  return normalized
    .replace(/<\/p>\s*<p[^>]*>/gi, "\n\n")
    .replace(/<br\s*\/?>/gi, "\n")
    .replace(/<\/(div|p|li|tr|h[1-6])>/gi, "\n")
    .replace(/<[^>]+>/g, "")
    .replace(/&nbsp;/g, " ")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&amp;/g, "&")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

function sanitizeDocumentTitle(value: string) {
  return (value || "")
    .replace(/\r/g, "")
    .split(/\n\s*(?:邮件正文建议|邮件正文|正文建议|建议正文)\s*[:：]/)[0]
    .replace(/\s+/g, " ")
    .trim()
    .slice(0, 120);
}

function extractRenderAssistTextSection(source: string, titles: string[]) {
  const text = (source || "").replace(/\r/g, "").trim();
  if (!text) return "";
  const titlePattern = titles.map((title) => title.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")).join("|");
  const boundaryTitles = [
    "邮件主题建议",
    "邮件主题",
    "主题建议",
    "建议标题",
    "邮件正文建议",
    "邮件正文",
    "正文建议",
    "建议正文",
  ]
    .map((title) => title.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
    .join("|");
  const pattern = new RegExp(
    `(?:${titlePattern})\\s*[:：]\\s*([\\s\\S]*?)(?=\\n\\s*(?:${boundaryTitles})\\s*[:：]?|$)`,
    "i",
  );
  return stripTemplateNoise(text.match(pattern)?.[1] || "");
}

function normalizeRenderAssistResponse(response: TemplateAssistResponse): TemplateAssistResponse {
  const summary = response.summary || "";
  const subjectFromSummary = extractRenderAssistTextSection(summary, ["邮件主题建议", "邮件主题", "主题建议", "建议标题"]);
  const bodyFromSummary = extractRenderAssistTextSection(summary, ["邮件正文建议", "邮件正文", "正文建议", "建议正文"]);
  const nextHtml = buildClientPreviewHtml(bodyFromSummary, {}) || response.html_template;
  return {
    ...response,
    summary: bodyFromSummary ? "已生成正文建议，可在下方编辑后预览或创建函件。" : response.summary,
    subject_template: subjectFromSummary || response.subject_template,
    html_template: nextHtml,
    plain_text: bodyFromSummary || htmlToEditableText(response.html_template),
  };
}

function buildTemplateLabelPreviewHtml(value: string, variables: TemplateVariable[]) {
  const normalized = value.trim();
  if (!normalized) return "";
  const rendered = renderTemplatePlaceholderLabels(normalized, variables);
  if (/<(p|div|br|ul|ol|li|table|tbody|thead|tr|td|th)\b/i.test(rendered)) {
    return rendered;
  }
  return rendered
    .split(/\n\s*\n/)
    .map((block) => block.trim())
    .filter(Boolean)
    .map((block) => `<p>${block.split("\n").map((line) => line.trim()).filter(Boolean).join("<br/>")}</p>`)
    .join("");
}

function stripTemplateNoise(value?: string) {
  return (value || "")
    .replace(/\r/g, "")
    .replace(/追问轮次[:：]\s*\d+/g, "")
    .replace(/请确认右侧草稿[^。！？!\n]*[。！？!]*/g, "")
    .replace(/请确认草稿[^。！？!\n]*[。！？!]*/g, "")
    .replace(/请您确认以上草稿[^。！？!\n]*[。！？!]*/g, "")
    .replace(/如无问题可直接保存[^。！？!\n]*[。！？!]*/g, "")
    .replace(/如需我继续生成[^。！？!\n]*[。！？!]*/g, "")
    .replace(/如需我直接继续[^。！？!\n]*[。！？!]*/g, "")
    .replace(/已更新草稿[^。！？!\n]*[。！？!]*/g, "")
    .replace(/草稿已经更新[^。！？!\n]*[。！？!]*/g, "")
    .replace(/^```(?:html)?\s*/i, "")
    .replace(/\s*```$/i, "")
    .trim();
}

function extractTemplateSection(source: string, title: string) {
  const escapedTitle = title.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const pattern = new RegExp(
    `(?:^|\\n)\\s*${escapedTitle}\\s*[:：]\\s*([\\s\\S]*?)(?=\\n\\s*(?:模板摘要|主题模板|HTML模板|HTML 模板|字段列表|AI 待确认问题|AI字段建议|AI 字段建议|待确认问题|建议字段)\\s*[:：]|$)`,
    "i",
  );
  const matched = source.match(pattern);
  return stripTemplateNoise(matched?.[1] || "");
}

function cleanTemplateSectionValue(value?: string) {
  return stripTemplateNoise(value)
    .replace(/^(模板摘要|主题模板|HTML模板|HTML 模板)\s*[:：]\s*/i, "")
    .trim();
}

function normalizeWhitespace(value?: string) {
  return (value || "").replace(/\s+/g, " ").trim();
}

function inferFollowUpQuestion(result: TemplateAssistResponse) {
  const candidates = [
    ...(result.follow_up_questions || []),
    result.summary,
    result.subject_template,
    result.html_template,
    result.plain_text,
  ]
    .map((item) => stripTemplateNoise(item))
    .filter(Boolean);

  return candidates.find((item) => isActualFollowUpQuestion(item) || hasQuestionLikeText(item)) || "";
}

function normalizeTemplateHtml(value?: string) {
  const normalized = cleanTemplateSectionValue(value);
  if (!normalized) return "";
  if (/^\d+$/.test(normalized)) return "";
  if (/^(请补充|请确认|如需|建议字段|待确认问题)/.test(normalized)) return "";
  if (hasQuestionLikeText(normalized)) return "";
  return normalized;
}

function normalizeTemplateAssistResult(result: TemplateAssistResponse): TemplateAssistResponse {
  const mergedText = [result.summary, result.subject_template, result.html_template, result.plain_text]
    .filter(Boolean)
    .join("\n");
  const htmlSection =
    extractTemplateSection(mergedText, "HTML 模板") ||
    extractTemplateSection(mergedText, "HTML模板");

  const summary =
    extractTemplateSection(mergedText, "模板摘要") ||
    cleanTemplateSectionValue(result.summary) ||
    cleanTemplateSectionValue(result.plain_text);
  const subjectTemplate =
    extractTemplateSection(mergedText, "主题模板") || cleanTemplateSectionValue(result.subject_template);
  const htmlTemplate =
    normalizeTemplateHtml(htmlSection) ||
    normalizeTemplateHtml(result.html_template) ||
    normalizeTemplateHtml(result.plain_text);

  const followUpQuestions = Array.from(
    new Set(
      (result.follow_up_questions || [])
        .map((item) => stripTemplateNoise(item))
        .filter((item) => isActualFollowUpQuestion(item)),
    ),
  );

  const normalizedResult = {
    ...result,
    summary: hasQuestionLikeText(summary) ? "" : normalizeWhitespace(summary),
    subject_template: hasQuestionLikeText(subjectTemplate) ? "" : normalizeWhitespace(subjectTemplate),
    html_template: htmlTemplate,
    follow_up_questions: followUpQuestions,
  };

  if (normalizedResult.follow_up_questions.length) {
    normalizedResult.summary = "";
    normalizedResult.subject_template = "";
    normalizedResult.html_template = "";
  }

  if (!normalizedResult.follow_up_questions.length) {
    const inferredQuestion = inferFollowUpQuestion(normalizedResult);
    if (inferredQuestion) {
      normalizedResult.follow_up_questions = [inferredQuestion];
      if (!extractTemplateSection(mergedText, "模板摘要")) normalizedResult.summary = "";
      if (!extractTemplateSection(mergedText, "主题模板")) normalizedResult.subject_template = "";
      if (!extractTemplateSection(mergedText, "HTML 模板") && !extractTemplateSection(mergedText, "HTML模板")) {
        normalizedResult.html_template = "";
      }
    }
  }

  return normalizedResult;
}

function isActualFollowUpQuestion(value: string) {
  const normalized = stripTemplateNoise(value);
  if (!normalized) return false;
  if (/已更新|草稿|可直接保存|模板已确认|右侧/.test(normalized)) return false;
  return (
    /[？?]/.test(normalized) ||
    /^(请补充|请确认|还需要|是否|需要你|请提供)/.test(normalized) ||
    /(能否|是否可以|是否可|可否|请问)/.test(normalized)
  );
}

function hasQuestionLikeText(value?: string) {
  const normalized = stripTemplateNoise(value);
  if (!normalized) return false;
  return (
    /[？?]/.test(normalized) ||
    /^(请补充|请确认|还需要|需要你|请提供|建议字段|待确认问题)/.test(normalized) ||
    /(面向对象|产物类型|语气风格|模板场景|必填字段|是否可以基于这些信息|生成并优化模板摘要|主题模板和HTML模板)/.test(normalized)
  );
}

function isMeaningfulHtmlTemplate(value?: string) {
  const normalized = normalizeTemplateHtml(value);
  if (!normalized) return false;
  if (normalized.length < 12) return false;
  return /[a-zA-Z\u4e00-\u9fa5]/.test(normalized);
}

function tabLabel(tab: TabKey) {
  const map: Record<TabKey, string> = {
    overview: "总览",
    mail: "邮箱",
    templates: "模板",
    documents: "函件",
  };
  return map[tab];
}
export function CommunicationCenterView() {
  const user = useAuthStore((state) => state.user);
  const isLP = user?.roles?.includes("LP") ?? false;

  const [activeTab, setActiveTab] = React.useState<TabKey>(isLP ? "documents" : "overview");
  const [loading, setLoading] = React.useState(false);
  const [message, setMessage] = React.useState("");
  const [actionLoading, setActionLoading] = React.useState<"" | "assist" | "create">("");

  const [summary, setSummary] = React.useState<DailySummary | null>(null);
  const [mailboxes, setMailboxes] = React.useState<MailboxAccount[]>([]);
  const [emails, setEmails] = React.useState<EmailItem[]>([]);
  const [selectedMailboxId, setSelectedMailboxId] = React.useState<number>(0);
  const [emailPage, setEmailPage] = React.useState({ page: 1, pageSize: 10, total: 0 });
  const [emailRefreshKey, setEmailRefreshKey] = React.useState(0);
  const [templates, setTemplates] = React.useState<TemplateItem[]>([]);
  const [templateCategoryFilter, setTemplateCategoryFilter] = React.useState("all");
  const [documents, setDocuments] = React.useState<DocumentItem[]>([]);
  const [lpRecipients, setLPRecipients] = React.useState<LPRecipient[]>([]);
  const [templateVersions, setTemplateVersions] = React.useState<Record<string, TemplateVersion[]>>({});
  const [openedTemplateId, setOpenedTemplateId] = React.useState("");

  const [renderAssist, setRenderAssist] = React.useState<TemplateAssistResponse | null>(null);
  const [documentDraftTitle, setDocumentDraftTitle] = React.useState("");
  const [documentDraftHtml, setDocumentDraftHtml] = React.useState("");
  const [selectedTemplateId, setSelectedTemplateId] = React.useState("");
  const [documentVariableValues, setDocumentVariableValues] = React.useState<Record<string, string>>({});
  const [templateActionLoading, setTemplateActionLoading] = React.useState<"" | "chat" | "import" | "save">("");
  const [templateDraft, setTemplateDraft] = React.useState<TemplateAssistResponse | null>(null);
  const [templateDraftVariables, setTemplateDraftVariables] = React.useState<TemplateVariable[]>([]);
  const [templateDraftMode, setTemplateDraftMode] = React.useState<"create" | "import">("create");
  const [templateChatMessages, setTemplateChatMessages] = React.useState<TemplateChatMessage[]>([]);
  const [templateChatInput, setTemplateChatInput] = React.useState("");
  const [templateConfirmed, setTemplateConfirmed] = React.useState(false);
  const [templateGoalSeed, setTemplateGoalSeed] = React.useState("");
  const [latestTemplateAssistantText, setLatestTemplateAssistantText] = React.useState("");
  const [templateWorkspace, setTemplateWorkspace] = React.useState<"draft" | "library">("draft");
  const [templateFieldPlacementState, setTemplateFieldPlacementState] = React.useState<TemplateFieldPlacementState | null>(null);
  const [templateForm, setTemplateForm] = React.useState<TemplateDraftFormState>({
    name: "",
    category: "notice",
    scene: "",
    usage_boundary: "",
    goal: "",
  });
  const [importTemplateFileName, setImportTemplateFileName] = React.useState("");
  const [importTemplateBase64, setImportTemplateBase64] = React.useState("");

  const usedTemplateDraftVariables = React.useMemo(() => {
    if (!templateDraft) return [] as TemplateVariable[];
    const usedNames = new Set([
      ...extractTemplateVariableNames(templateDraft.subject_template),
      ...extractTemplateVariableNames(templateDraft.html_template),
    ]);
    return templateDraftVariables.filter((item) => usedNames.has(item.variable_name));
  }, [templateDraft, templateDraftVariables]);

  const effectiveSummary = React.useMemo(() => {
    if (!summary) return null;
    const unread = emails.filter((item) => !item.is_read).length;
    const needsReply = emails.filter((item) => item.needs_reply).length;
    const priorityMap = new Map<string, EmailItem>();
    [...(summary.top_priority || []), ...emails.filter((item) => item.is_starred)].forEach((item) => {
      const existing = priorityMap.get(item.email_id);
      if (!existing || (item.priority_score ?? 0) > (existing.priority_score ?? 0) || item.is_starred) {
        priorityMap.set(item.email_id, item);
      }
    });
    const priorityTotal = priorityMap.size;
    const topPriority = Array.from(priorityMap.values())
      .sort((a, b) => {
        if (a.is_starred !== b.is_starred) return a.is_starred ? -1 : 1;
        return (b.priority_score ?? 0) - (a.priority_score ?? 0);
      })
      .slice(0, 5);
    return {
      ...summary,
      unread,
      needs_reply: needsReply,
      top_priority: topPriority,
      priority_total: priorityTotal,
      daily_brief:
        emails.length > 0
          ? `当前收件箱共 ${emails.length} 封邮件，其中 ${unread} 封未读，${needsReply} 封需要回复，${priorityTotal} 封属于今日重点。`
          : summary.daily_brief,
    };
  }, [emails, summary]);

  function applyFieldPlacementToDraft(variableName: string, variableLabel: string, placement: TemplateFieldPlacement) {
    if (!templateDraft) return false;

    const nextVariable: TemplateVariable = {
      variable_name: variableName,
      variable_label: variableLabel,
      variable_type: "string",
      is_required: true,
      default_value: "",
    };

    const nextSubject =
      placement === "subject" || placement === "both"
        ? injectFieldIntoSubjectTemplate(templateDraft.subject_template || "", variableName)
        : templateDraft.subject_template || "";

    const nextHtml =
      placement === "html" || placement === "both"
        ? injectFieldIntoHtmlTemplate(templateDraft.html_template || "", variableName, variableLabel)
        : templateDraft.html_template || "";

    setTemplateDraft((prev) =>
      prev
        ? {
            ...prev,
            subject_template: nextSubject,
            html_template: nextHtml,
            follow_up_questions: [],
          }
        : prev,
    );
    mergeTemplateVariables([nextVariable]);
    setTemplateConfirmed(true);
    setTemplateFieldPlacementState(null);
    const placementLabel =
      placement === "both" ? "主题和正文" : placement === "subject" ? "主题" : "正文";
    const assistantText = `已新增字段“${variableLabel}”，并放入${placementLabel}。右侧草稿已更新。`;
    setLatestTemplateAssistantText(assistantText);
    appendTemplateChat("assistant", assistantText);
    setMessage(`字段“${variableLabel}”已加入${placementLabel}`);
    return true;
  }

  const [mailboxForm, setMailboxForm] = React.useState<MailboxPayload>({
    name: "",
    mailbox_type: "shared",
    email_address: "",
    auth_type: "password",
    smtp_host: "",
    smtp_port: 587,
    imap_host: "",
    imap_port: 993,
    username: "",
    password: "",
    oauth_provider: "",
    signature_html: "<p>Regards</p>",
    is_default_sender: false,
  });

  const [documentForm, setDocumentForm] = React.useState<DocumentFormState>({
    sender_mailbox_id: 0,
    lp_user_id: "",
    recipient_name: "",
    recipient_email: "",
    title: "",
    fund_name: "",
    lp_name: "",
    notice_date: new Date().toLocaleDateString("en-CA"),
  });
  const [additionalRecipientEmails, setAdditionalRecipientEmails] = React.useState<string[]>([]);

  const loadData = React.useCallback(async () => {
    setLoading(true);
    try {
      const [mailboxList, templatePage, documentPage, lpList] = await Promise.all([
        communicationApi.listMailboxes(),
        isLP ? Promise.resolve({ list: [], pagination: { page: 1, page_size: 1, total: 0 } }) : communicationApi.listTemplates(),
        communicationApi.listDocuments(),
        isLP ? Promise.resolve([] as LPRecipient[]) : communicationApi.listLPRecipients(),
      ]);

      setMailboxes(mailboxList);
      const sendableMailboxList = mailboxList.filter(canUseMailboxAsSender);
      const defaultSender =
        sendableMailboxList.find((item) => item.mailbox_id === selectedMailboxId) ??
        sendableMailboxList.find((item) => item.email_address.includes("@139.com")) ??
        sendableMailboxList.find((item) => item.imap_host && !item.imap_host.includes("example.com")) ??
        sendableMailboxList.find((item) => item.is_default_sender) ??
        sendableMailboxList[0];
      const nextMailboxId = defaultSender?.mailbox_id || 0;
      setSelectedMailboxId(nextMailboxId);
      const emailResult = nextMailboxId
        ? await communicationApi.listEmails({ mailbox_id: nextMailboxId, page: emailPage.page, page_size: emailPage.pageSize, sort_by: "received_at" })
        : await communicationApi.listEmails({ page: emailPage.page, page_size: emailPage.pageSize, sort_by: "received_at" });
      setEmails(emailResult.list);
      setEmailPage((prev) => ({ ...prev, total: emailResult.pagination.total }));
      setTemplates(templatePage.list);
      setDocuments(documentPage.list);
      setLPRecipients(lpList);

      const defaultLP = lpList[0];

      setDocumentForm((prev) => ({
        ...prev,
        sender_mailbox_id: prev.sender_mailbox_id || defaultSender?.mailbox_id || 0,
        recipient_name: prev.recipient_name || defaultLP?.name || "",
        recipient_email: prev.recipient_email || defaultLP?.email || "",
        lp_name: prev.lp_name || defaultLP?.name || "",
      }));

      if (defaultSender) {
        try {
          setSummary(await communicationApi.emailSummary(defaultSender.mailbox_id));
        } catch {
          setSummary(null);
        }
      } else {
        setSummary(null);
      }
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "加载通讯管理数据失败");
    } finally {
      setLoading(false);
    }
  }, [emailPage.page, emailPage.pageSize, emailRefreshKey, isLP, selectedMailboxId]);

  React.useEffect(() => {
    void loadData();
  }, [loadData]);

  const selectableTemplates = React.useMemo(
    () => templates.filter((item) => item.status !== "deleted" && item.template_source !== "published_public"),
    [templates],
  );

  const libraryTemplates = React.useMemo(
    () => templates.filter((item) => templateCategoryFilter === "all" || item.category === templateCategoryFilter),
    [templateCategoryFilter, templates],
  );

  const selectedTemplate = React.useMemo(
    () => selectableTemplates.find((item) => item.template_id === selectedTemplateId),
    [selectableTemplates, selectedTemplateId],
  );

  const extraDocumentVariables = React.useMemo(
    () => (selectedTemplate?.variables || []).filter((item) => !fixedDocumentVariableFields[item.variable_name]),
    [selectedTemplate],
  );

  React.useEffect(() => {
    setRenderAssist(null);
    setDocumentDraftTitle("");
    setDocumentDraftHtml("");
    setDocumentVariableValues((prev) => {
      if (!selectedTemplate) return prev;
      const next: Record<string, string> = {};
      selectedTemplate.variables.forEach((item) => {
        if (!fixedDocumentVariableFields[item.variable_name]) {
          next[item.variable_name] = prev[item.variable_name] ?? item.default_value ?? "";
        }
      });
      return next;
    });
  }, [selectedTemplateId, selectedTemplate]);

  React.useEffect(() => {
    if (!renderAssist) return;
    const timer = window.setTimeout(() => {
      document.getElementById("communication-render-assist")?.scrollIntoView({ behavior: "smooth", block: "start" });
    }, 50);
    return () => window.clearTimeout(timer);
  }, [renderAssist]);

  function normalizeVariableName(value: string) {
    return value
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9\u4e00-\u9fa5]+/g, "_")
      .replace(/^_+|_+$/g, "");
  }

  function mergeTemplateVariables(nextVariables: TemplateVariable[]) {
    setTemplateDraftVariables((prev) => {
      const variableMap = new Map<string, TemplateVariable>();
      [...prev, ...nextVariables].forEach((item) => {
        const key = normalizeVariableName(item.variable_name);
        if (!key) return;
        variableMap.set(key, {
          variable_name: key,
          variable_label: item.variable_label || item.variable_name || key,
          variable_type: item.variable_type || "string",
          is_required: item.is_required ?? true,
          default_value: item.default_value || "",
        });
      });
      return Array.from(variableMap.values());
    });
  }

  async function readFileAsBase64(file: File) {
    return await new Promise<string>((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        const result = String(reader.result || "");
        const parts = result.split(",");
        resolve(parts[1] || "");
      };
      reader.onerror = () => reject(reader.error || new Error("璇诲彇鏂囦欢澶辫触"));
      reader.readAsDataURL(file);
    });
  }

  function buildTemplateAssistantMessage(result: TemplateAssistResponse, hasConfirmedDraft: boolean) {
  if (result.follow_up_questions.length && !hasConfirmedDraft) {
    return result.follow_up_questions[0];
  }
  if (hasConfirmedDraft) {
    return "草稿已更新到右侧。如需继续修改，直接补充你的要求即可。";
  }
  return "AI 已形成一版可用草稿，右侧已更新。如需调整，继续输入修改意见即可。";
}

function isTemplateDraftReady(result: TemplateAssistResponse) {
  const hasSummary = Boolean(result.summary?.trim());
  const hasSubject = Boolean(result.subject_template?.trim());
  const hasHtml = isMeaningfulHtmlTemplate(result.html_template);
  const hasRealQuestions = (result.follow_up_questions || []).some((item) => isActualFollowUpQuestion(item));
  const summaryStillQuestioning = hasQuestionLikeText(result.summary);
  const explicitConfirmed = /草稿已更新|可用草稿|已形成.*草稿|右侧已更新|可直接保存|模板已确认|请确认草稿/.test(
    [result.summary, result.subject_template, result.html_template, result.plain_text, ...(result.follow_up_questions || [])]
      .filter(Boolean)
      .join(" "),
  );
  return hasSummary && hasSubject && hasHtml && (explicitConfirmed || (!hasRealQuestions && !summaryStillQuestioning));
}

function shouldRenderTemplateDraft(
  draft: TemplateAssistResponse | null,
  confirmed: boolean,
  latestAssistantText: string,
) {
  if (!draft) return false;
  const ready = isTemplateDraftReady(draft);
  if (!ready) return false;
  return confirmed || /草稿已更新|可用草稿|模板已确认|右侧已更新/.test(latestAssistantText) || ready;
}
  function resetTemplateConversation(mode: "create" | "import") {
    setTemplateWorkspace("draft");
    setTemplateDraftMode(mode);
    setTemplateDraft(null);
    setTemplateDraftVariables([]);
    setTemplateChatMessages([]);
    setTemplateChatInput("");
    setTemplateConfirmed(false);
    setTemplateGoalSeed("");
    setLatestTemplateAssistantText("");
    setTemplateFieldPlacementState(null);
    if (mode === "import") {
      setTemplateForm((prev) => ({ ...prev, goal: "" }));
    }
  }

  function appendTemplateChat(role: "user" | "assistant", text: string) {
    if (!text.trim()) return;
    setTemplateChatMessages((prev) => [
      ...prev,
      {
        id: `${role}-${Date.now()}-${prev.length}`,
        role,
        text: text.trim(),
      },
    ]);
  }

  function buildTemplateConversationContext(userMessage: string) {
    const recentMessages = templateChatMessages.slice(-6).map((item) => `${item.role === "user" ? "用户" : "AI"}：${item.text}`);
    const lastAssistantQuestion = [...templateChatMessages]
      .reverse()
      .find((item) => item.role === "assistant" && /[？?]$/.test(item.text));

    const pairedUserMessage = lastAssistantQuestion
      ? `上一轮问题：${lastAssistantQuestion.text}\n本轮回答：${userMessage}`
      : userMessage;

    return {
      latestUserMessage: pairedUserMessage,
      currentSummary: [templateDraft?.summary, ...recentMessages].filter(Boolean).join("\n"),
    };
  }

  async function saveMailbox() {
    await communicationApi.createMailbox(mailboxForm);
    setMessage("邮箱配置已保存");
    setMailboxForm((prev) => ({
      ...prev,
      name: "",
      email_address: "",
      smtp_host: "",
      smtp_port: 587,
      imap_host: "",
      imap_port: 993,
      username: "",
      password: "",
    }));
    await loadData();
  }

  async function syncMailbox(id: number) {
    const result = await communicationApi.syncMailbox(id);
    const rawMessage = String(result.message ?? "");
    const match = rawMessage.match(/(\d+) new email/);
    setMessage(match ? (match[1] === "0" ? "邮箱同步完成，暂无新邮件" : `邮箱同步完成，新增 ${match[1]} 封邮件`) : rawMessage || "邮箱同步完成");
    setSelectedMailboxId(id);
    setEmailPage((prev) => ({ ...prev, page: 1 }));
    setEmailRefreshKey((prev) => prev + 1);
  }

  async function deleteMailbox(id: number) {
    if (!window.confirm("删除该邮箱配置及其本地同步邮件？此操作不可恢复。")) return;
    await communicationApi.deleteMailbox(id);
    if (selectedMailboxId === id) setSelectedMailboxId(0);
    setMessage("邮箱配置已删除");
    await loadData();
  }

  async function connectOutlookMailbox(id: number) {
    const result = await communicationApi.beginOutlookAuthorization(id);
    if (!result.authorization_url) {
      throw new Error("未获取到 Microsoft 授权地址");
    }
    window.location.assign(result.authorization_url);
  }

  async function markRead(emailId: string) {
    await communicationApi.markRead(emailId);
    setEmails((prev) => prev.map((item) => (item.email_id === emailId ? { ...item, is_read: true } : item)));
    setSummary((prev) =>
      prev
        ? {
            ...prev,
            unread: Math.max(0, prev.unread - (prev.top_priority.find((item) => item.email_id === emailId && !item.is_read) ? 1 : 0)),
            top_priority: prev.top_priority.map((item) => (item.email_id === emailId ? { ...item, is_read: true } : item)),
          }
        : prev,
    );
  }

  async function starEmail(emailId: string) {
    const result = await communicationApi.starEmail(emailId);
    setEmails((prev) => prev.map((item) => (item.email_id === emailId ? { ...item, is_starred: result.is_starred } : item)));
    setSummary((prev) =>
      prev
        ? {
            ...prev,
            top_priority: prev.top_priority.map((item) => (item.email_id === emailId ? { ...item, is_starred: result.is_starred } : item)),
          }
        : prev,
    );
  }

  async function deleteEmail(emailId: string) {
    if (!window.confirm("删除这封本地收件箱邮件？此操作不会删除外部邮箱中的原邮件。")) return;
    await communicationApi.deleteEmail(emailId);
    setEmails((prev) => prev.filter((item) => item.email_id !== emailId));
    setEmailRefreshKey((value) => value + 1);
    setMessage("邮件已删除");
  }

  async function toggleTemplateHistory(templateId: string) {
    if (openedTemplateId === templateId) {
      setOpenedTemplateId("");
      return;
    }
    if (!templateVersions[templateId]) {
      const versions = await communicationApi.listTemplateVersions(templateId);
      setTemplateVersions((prev) => ({ ...prev, [templateId]: versions }));
    }
    setOpenedTemplateId(templateId);
  }

  function chooseLPRecipient(userId: string) {
    const target = lpRecipients.find((item) => item.user_id === userId);
    setDocumentForm((prev) => ({
      ...prev,
      lp_user_id: userId,
      recipient_name: target?.name || prev.recipient_name,
      recipient_email: target?.email || prev.recipient_email,
      lp_name: target?.name || prev.lp_name,
    }));
  }

  function buildDocumentVariables() {
    const variables = new Map<string, string>();
    (selectedTemplate?.variables || []).forEach((item) => {
      variables.set(item.variable_name, deriveDocumentVariableValue(item.variable_name, documentForm, documentVariableValues));
    });
    if (!variables.has("fund_name")) variables.set("fund_name", documentForm.fund_name);
    if (!variables.has("lp_name")) variables.set("lp_name", documentForm.lp_name);
    if (!variables.has("notice_date")) variables.set("notice_date", documentForm.notice_date);
    return Object.fromEntries(variables.entries());
  }

  function additionalRecipients() {
    return toRecipientRefs(additionalRecipientEmails);
  }

  function getEffectiveDocumentTitle() {
    return sanitizeDocumentTitle(documentDraftTitle || documentForm.title);
  }

  function getEffectiveDocumentHtml() {
    return documentDraftHtml.trim();
  }

  function getMissingRequiredDocumentVariables() {
    return (selectedTemplate?.variables || []).filter((item) => {
      if (!item.is_required) return false;
      return !deriveDocumentVariableValue(item.variable_name, documentForm, documentVariableValues).trim();
    });
  }

  async function askRenderAssist() {
    if (!selectedTemplateId) {
      setMessage("请先选择模板，再生成正文建议");
      return;
    }
    const missing = getMissingRequiredDocumentVariables();
    if (missing.length) {
      setMessage(`请先补全模板必填字段：${missing.map(getVariableLabel).join("、")}`);
      return;
    }
    const rawResponse = await communicationApi.assistTemplateRender({
      template_name: selectedTemplate?.name,
      template_scene: selectedTemplate?.scene,
      recipient_type: documentForm.lp_user_id ? "LP" : "external",
      business_goal: documentForm.title || "生成正式对外通知",
      known_values: buildDocumentVariables(),
    });
    const response = normalizeRenderAssistResponse(rawResponse);
    setRenderAssist(response);
    if (response.subject_template) {
      // The operator-entered title is the source of truth. AI only fills it when blank.
      setDocumentDraftTitle(documentForm.title.trim() || response.subject_template);
      if (!documentForm.title) {
        setDocumentForm((prev) => ({ ...prev, title: response.subject_template }));
      }
    }
    if (response.html_template) {
      setDocumentDraftHtml(response.html_template);
    }
    setMessage("AI 已生成正文建议，可直接编辑后创建函件");
  }

  async function createDocument() {
    if (!selectedTemplateId) {
      setMessage("请先选择模板，再创建函件");
      return;
    }
    const missing = getMissingRequiredDocumentVariables();
    if (missing.length) {
      setMessage(`请先补全模板必填字段：${missing.map(getVariableLabel).join("、")}`);
      return;
    }
    const result = await communicationApi.createDocument({
      template_id: selectedTemplateId,
      recipient_name: documentForm.recipient_name,
      recipient_email: documentForm.recipient_email,
      cc_recipients: additionalRecipients(),
      sender_mailbox_id: documentForm.sender_mailbox_id,
      title: getEffectiveDocumentTitle(),
      html_content: getEffectiveDocumentHtml(),
      document_type: "notice",
      visible_user_ids: documentForm.lp_user_id ? [documentForm.lp_user_id] : [],
      variables: buildDocumentVariables(),
    });
    setMessage(result.docx_file_name ? "函件实例已创建" : "函件实例已创建，但未配置正式 DOCX");
    await loadData();
  }

  async function runDocumentAction(nextLoading: "assist" | "create", loadingMessage: string, action: () => Promise<void>) {
    if (actionLoading) return;
    setActionLoading(nextLoading);
    setMessage(loadingMessage);
    try {
      await action();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "操作失败，请检查必填信息后重试");
    } finally {
      setActionLoading("");
    }
  }

  async function submitDocument(id: string) {
    const result = await communicationApi.submitDocument(id);
    setMessage(`函件已提审，审批单号：${String(result.approval_id ?? "")}`);
    await loadData();
  }

  async function sendDocument(id: string) {
    const result = await communicationApi.sendDocument(id);
    setMessage(`函件已发送，发送时间：${String(result.sent_at ?? "")}`);
    await loadData();
  }

  async function downloadDocument(item: DocumentItem) {
    if (!item.docx_file_name) {
      setMessage("褰撳墠妯℃澘灏氭湭閰嶇疆姝ｅ紡 DOCX");
      return;
    }
    await communicationApi.downloadDocument(item.document_id, item.docx_file_name || `${item.title}.docx`);
  }

  async function draftTemplateFromChat() {
    try {
      const latestMessage = templateChatInput.trim();
      if (!templateForm.name.trim()) {
        setMessage("璇峰厛濉啓妯℃澘鍚嶇О");
        return;
      }
      if (!latestMessage) {
        setMessage("璇峰厛杈撳叆妯℃澘闇€姹傛垨淇敼鎰忚");
        return;
      }

      appendTemplateChat("user", latestMessage);

      if (templateDraft) {
        if (templateFieldPlacementState) {
          const placement = inferPlacementFromMessage(latestMessage);
          if (placement) {
            setTemplateChatInput("");
            applyFieldPlacementToDraft(
              templateFieldPlacementState.variableName,
              templateFieldPlacementState.variableLabel,
              placement,
            );
            return;
          }

          const nextAttempt = templateFieldPlacementState.attempt + 1;
          const nextState = { ...templateFieldPlacementState, attempt: nextAttempt };
          setTemplateFieldPlacementState(nextState);
          const clarification = buildPlacementClarificationMessage(nextState.variableLabel, nextAttempt);
          setLatestTemplateAssistantText(clarification);
          appendTemplateChat("assistant", clarification);
          setTemplateChatInput("");
          setMessage(`字段“${nextState.variableLabel}”的落位仍不明确，等待你选择位置`);
          return;
        }

        const fieldIntent = extractFieldIntent(latestMessage);
        if (fieldIntent) {
          const placement = inferPlacementFromMessage(latestMessage);
          if (placement) {
            setTemplateChatInput("");
            applyFieldPlacementToDraft(fieldIntent.variableName, fieldIntent.variableLabel, placement);
            return;
          }

          const pendingState: TemplateFieldPlacementState = {
            variableName: fieldIntent.variableName,
            variableLabel: fieldIntent.variableLabel,
            attempt: 1,
          };
          setTemplateFieldPlacementState(pendingState);
          const clarification = buildPlacementClarificationMessage(fieldIntent.variableLabel, 1);
          setLatestTemplateAssistantText(clarification);
          appendTemplateChat("assistant", clarification);
          setTemplateChatInput("");
          setMessage(`字段“${fieldIntent.variableLabel}”需要先确认放在主题、正文还是两者都放`);
          return;
        }
      }

      const goal = templateGoalSeed || templateForm.goal.trim() || latestMessage;
      const chatContext = buildTemplateConversationContext(latestMessage);
      setLatestTemplateAssistantText("AI 正在生成模板草稿，请稍候...");
      setMessage("已发送给 AI，正在生成模板草稿...");
      const rawResult = await communicationApi.draftTemplateFromChat({
        scene: templateForm.scene.trim(),
        goal,
        current_summary: chatContext.currentSummary,
        latest_user_message: chatContext.latestUserMessage,
        known_fields: templateDraftVariables,
      });
      const result = normalizeTemplateAssistResult(rawResult);
      if (!templateGoalSeed) {
        setTemplateGoalSeed(goal);
        setTemplateForm((prev) => ({ ...prev, goal }));
      }
      const hasConfirmedDraft = isTemplateDraftReady(result);
      setTemplateDraft((prev) => {
        if (hasConfirmedDraft) return result;
        if (templateConfirmed && prev) {
          return {
            ...prev,
            follow_up_questions: result.follow_up_questions,
            suggested_fields: result.suggested_fields?.length ? result.suggested_fields : prev.suggested_fields,
            plain_text: result.plain_text || prev.plain_text,
          };
        }
        return result;
      });
      mergeTemplateVariables(result.suggested_fields || []);
      setTemplateConfirmed(hasConfirmedDraft);
      const assistantText = buildTemplateAssistantMessage(result, hasConfirmedDraft);
      setLatestTemplateAssistantText(assistantText);
      const lastAssistantText = [...templateChatMessages].reverse().find((item) => item.role === "assistant")?.text;
      if (assistantText !== lastAssistantText) {
        appendTemplateChat("assistant", assistantText);
      }
      if (hasConfirmedDraft) {
        setTemplateFieldPlacementState(null);
      }
      setTemplateChatInput("");
      setMessage(hasConfirmedDraft ? "AI 已响应，草稿已更新到右侧" : "AI 已响应，正在继续确认关键信息");
    } catch (error) {
      const text = error instanceof Error ? error.message : "妯℃澘鐢熸垚澶辫触";
      setLatestTemplateAssistantText(`请求失败：${text}`);
      appendTemplateChat("assistant", `请求失败：${text}`);
      setMessage(text);
    }
  }

  async function draftTemplateFromImport() {
    try {
      if (!templateForm.name.trim()) {
        setMessage("璇峰厛濉啓妯℃澘鍚嶇О");
        return;
      }
      if (!importTemplateBase64 || !importTemplateFileName) {
        setMessage("璇峰厛涓婁紶 DOCX 妯℃澘鏂囦欢");
        return;
      }
      setLatestTemplateAssistantText("AI 正在解析并整理模板草稿，请稍候...");
      setMessage("已提交 DOCX，正在解析模板...");
      const rawResult = await communicationApi.draftTemplateFromImport({
        file_name: importTemplateFileName,
        scene: templateForm.scene.trim(),
        docx_template_base64: importTemplateBase64,
      });
      const result = normalizeTemplateAssistResult(rawResult);
      const hasConfirmedDraft = isTemplateDraftReady(result);
      setTemplateDraft((prev) => {
        if (hasConfirmedDraft) return result;
        if (templateConfirmed && prev) {
          return {
            ...prev,
            follow_up_questions: result.follow_up_questions,
            suggested_fields: result.suggested_fields?.length ? result.suggested_fields : prev.suggested_fields,
            plain_text: result.plain_text || prev.plain_text,
          };
        }
        return result;
      });
      mergeTemplateVariables(result.suggested_fields || []);
      setTemplateConfirmed(hasConfirmedDraft);
      const assistantText = buildTemplateAssistantMessage(result, hasConfirmedDraft);
      setLatestTemplateAssistantText(assistantText);
      const lastAssistantText = [...templateChatMessages].reverse().find((item) => item.role === "assistant")?.text;
      if (assistantText !== lastAssistantText) {
        appendTemplateChat("assistant", assistantText);
      }
      setTemplateChatInput("");
      setMessage(hasConfirmedDraft ? "模板已解析并生成草稿" : "模板已解析，正在继续确认关键信息");
    } catch (error) {
      const text = error instanceof Error ? error.message : "妯℃澘瑙ｆ瀽澶辫触";
      setLatestTemplateAssistantText(`请求失败：${text}`);
      appendTemplateChat("assistant", `请求失败：${text}`);
      setMessage(text);
    }
  }

  async function saveTemplateDraft() {
    try {
      if (!templateDraft) {
        setMessage("请先生成模板草稿");
        return;
      }
      if (!isTemplateDraftReady(templateDraft)) {
        setMessage("当前草稿还不完整，请继续和 AI 补充模板摘要、主题模板和 HTML 模板后再保存");
        return;
      }
      if (!templateForm.name.trim()) {
        setMessage("请先填写模板名称");
        return;
      }
      setMessage("正在保存模板...");
      await communicationApi.createTemplate({
        name: templateForm.name.trim(),
        category: templateForm.category.trim() || "notice",
        scene: templateForm.scene.trim(),
        usage_boundary: templateForm.usage_boundary.trim(),
        parse_status: templateDraftMode === "import" ? "parsed" : "drafted",
        import_file_name: templateDraftMode === "import" ? importTemplateFileName : undefined,
        subject_template: templateDraft.subject_template,
        html_content: templateDraft.html_template,
        docx_template_name: templateDraftMode === "import" ? importTemplateFileName : undefined,
        docx_template_base64: templateDraftMode === "import" ? importTemplateBase64 : undefined,
        variables: templateDraftVariables,
      });
      setMessage("模板已保存，并已加入我的私人模板");
      setTemplateDraft(null);
      setTemplateDraftVariables([]);
      setTemplateChatMessages([]);
      setTemplateChatInput("");
      setTemplateConfirmed(false);
      setTemplateGoalSeed("");
      setLatestTemplateAssistantText("");
      setTemplateForm({
        name: "",
        category: "notice",
        scene: "",
        usage_boundary: "",
        goal: "",
      });
      setImportTemplateBase64("");
      setImportTemplateFileName("");
      await loadData();
    } catch (error) {
      const text = error instanceof Error ? error.message : "保存模板失败";
      setMessage(text);
    }
  }

  return (
    <section className="communication-center space-y-5">
      <header className="rounded-[24px] bg-white p-6 shadow-sm ring-1 ring-black/5">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <h1 className="text-[1.6rem] font-black text-slate-900">通讯管理</h1>
            <p className="mt-2 text-sm text-slate-500">邮箱、模板、函件和对外沟通记录统一在这里处理。</p>
          </div>
          <div className="rounded-full bg-amber-50 px-4 py-2 text-xs font-semibold text-amber-700">
            {effectiveSummary?.daily_brief || summary?.daily_brief || "今日邮件摘要暂未加载"}
          </div>
        </div>
        {!isLP ? (
          <div className={"communication-overview-collapse " + (activeTab === "overview" ? "is-open" : "")}>
            <div className="grid gap-3 md:grid-cols-3">
                <SummaryBadge label="重点邮件" value={String(effectiveSummary?.priority_total ?? effectiveSummary?.top_priority?.length ?? summary?.top_priority?.length ?? 0)} tone="amber" />
                <SummaryBadge label="未读邮件" value={String(effectiveSummary?.unread ?? summary?.unread ?? 0)} tone="blue" />
                <SummaryBadge label="需回复邮件" value={String(effectiveSummary?.needs_reply ?? summary?.needs_reply ?? 0)} tone="emerald" />
            </div>
          </div>
        ) : null}
        {message ? <p className="mt-4 rounded-xl bg-slate-50 px-4 py-3 text-sm text-slate-600">{message}</p> : null}
      </header>

      {!isLP ? (
        <div className={"communication-overview-collapse " + (activeTab === "overview" ? "is-open" : "")}>
          <div className="grid gap-4 md:grid-cols-4">
              <MetricCard label="邮箱数量" value={String(mailboxes.length)} hint="共享邮箱与发件邮箱" />
              <MetricCard label="模板数量" value={String(selectableTemplates.length)} hint="当前可选模板" />
              <MetricCard label="函件数量" value={String(documents.length)} hint="已创建的函件实例" />
              <MetricCard label="未读/待回" value={`${effectiveSummary?.unread ?? summary?.unread ?? 0}/${effectiveSummary?.needs_reply ?? summary?.needs_reply ?? 0}`} hint="邮件侧统计" />
          </div>
        </div>
      ) : null}

      <div className="flex flex-wrap gap-2">
        {(isLP ? (["documents"] as TabKey[]) : (["overview", "mail", "templates", "documents"] as TabKey[])).map((tab) => (
          <button
            key={tab}
            className={`communication-tab-button rounded-full px-4 py-2 text-sm font-semibold ${activeTab === tab ? "bg-slate-900 text-white" : "bg-white text-slate-600 ring-1 ring-slate-200"}`}
            onClick={() => setActiveTab(tab)}
          >
            {tabLabel(tab)}
          </button>
        ))}
      </div>

      {loading ? <EmptyHint text="正在加载通讯管理数据..." /> : null}

      {!isLP && activeTab === "overview" ? (
        <div className="grid gap-4 lg:grid-cols-[1.05fr_0.95fr]">
          <Panel title="今日重点邮件">
            {effectiveSummary?.top_priority?.length ? (
              <div className="space-y-3">
                {effectiveSummary.top_priority.map((item) => (
                  <article key={item.email_id} className="rounded-2xl border border-slate-100 p-4">
                    <div className="flex items-center justify-between gap-3">
                      <strong className="text-sm text-slate-900">{item.subject}</strong>
                      <span className="rounded-full bg-amber-100 px-2 py-1 text-[11px] font-bold text-amber-700">{item.priority_label}</span>
                    </div>
                    <p className="mt-2 text-sm text-slate-500">{item.summary || item.preview}</p>
                  </article>
                ))}
              </div>
            ) : (
              <EmptyHint text="当前没有重点邮件。" />
            )}
          </Panel>

          <Panel title="通讯中心概览">
            <div className="space-y-3 text-sm text-slate-600">
              <InfoRow label="发件邮箱" value={mailboxes[0]?.email_address || "-"} />
              <InfoRow label="最近模板" value={templates[0]?.name || "-"} />
              <InfoRow label="最近函件" value={documents[0]?.title || "-"} />
              <InfoRow label="LP 收件对象" value={String(lpRecipients.length)} />
            </div>
          </Panel>
        </div>
      ) : null}

      {!isLP && activeTab === "mail" ? (
        <div className="grid gap-4 lg:grid-cols-[0.85fr_1.35fr]">
          <Panel title="邮箱配置">
            <div className="grid gap-3">
              <Input label="名称" value={mailboxForm.name} onChange={(value) => setMailboxForm((prev) => ({ ...prev, name: value }))} />
              <Input label="邮箱地址" value={mailboxForm.email_address} onChange={(value) => setMailboxForm((prev) => ({ ...prev, email_address: value }))} />
              <label className="grid gap-2 text-sm font-semibold text-slate-700">
                <span>连接方式</span>
                <select
                  className="communication-field rounded-2xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-slate-400"
                  value={mailboxForm.auth_type}
                  onChange={(event) => setMailboxForm((prev) => ({
                    ...prev,
                    auth_type: event.target.value,
                    oauth_provider: event.target.value === "oauth" ? "microsoft_graph" : "",
                    smtp_host: event.target.value === "oauth" ? "" : prev.smtp_host,
                    imap_host: event.target.value === "oauth" ? "" : prev.imap_host,
                    password: event.target.value === "oauth" ? "" : prev.password,
                  }))}
                >
                  <option value="password">IMAP / SMTP 密码或授权码</option>
                  <option value="oauth">Outlook Microsoft OAuth</option>
                </select>
              </label>
              {mailboxForm.auth_type === "oauth" ? (
                <p className="rounded-2xl bg-blue-50 px-4 py-3 text-xs leading-6 text-blue-800">
                  Outlook 使用 Microsoft 登录授权，不填写 SMTP、IMAP、用户名或密码。保存后，在下方对应邮箱卡片点击“绑定 Outlook”。
                </p>
              ) : (
                <>
                  <Input label="SMTP Host" value={mailboxForm.smtp_host} onChange={(value) => setMailboxForm((prev) => ({ ...prev, smtp_host: value }))} />
                  <Input label="SMTP Port" type="number" value={mailboxForm.smtp_port} onChange={(value) => setMailboxForm((prev) => ({ ...prev, smtp_port: Number(value) || 587 }))} />
                  <Input label="IMAP Host" value={mailboxForm.imap_host} onChange={(value) => setMailboxForm((prev) => ({ ...prev, imap_host: value }))} />
                  <Input label="IMAP Port" type="number" value={mailboxForm.imap_port} onChange={(value) => setMailboxForm((prev) => ({ ...prev, imap_port: Number(value) || 993 }))} />
                  <Input label="用户名" value={mailboxForm.username} onChange={(value) => setMailboxForm((prev) => ({ ...prev, username: value }))} />
                  <Input label="密码/授权码" type="password" value={mailboxForm.password} onChange={(value) => setMailboxForm((prev) => ({ ...prev, password: value }))} />
                </>
              )}
            </div>
            <p className="mt-3 rounded-2xl bg-sky-50 px-4 py-3 text-xs leading-6 text-sky-800">
              139 邮箱配置提示：SMTP 填 smtp.139.com:465；IMAP 优先试 imap.139.com:993，如果 993 出现 TLS 握手失败，改用 imap.139.com:143；密码请填邮箱后台生成的授权码。
            </p>
            <button className="mt-4 rounded-full bg-slate-900 px-4 py-2 text-sm font-semibold text-white" onClick={() => void saveMailbox()}>
              保存邮箱配置
            </button>
            <div className="mt-5 space-y-3">
              {mailboxes.map((item) => (
                <article key={item.mailbox_id} className="rounded-2xl border border-slate-100 p-4">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <div className="font-semibold text-slate-900">{item.name}</div>
                      <div className="mt-1 text-xs text-slate-500">{item.email_address}</div>
                    </div>
                    {selectedMailboxId === item.mailbox_id ? (
                      <span className="rounded-full bg-emerald-50 px-3 py-1 text-[11px] font-bold text-emerald-700">当前收件箱</span>
                    ) : (
                      <button
                        className="rounded-full bg-slate-100 px-3 py-1 text-[11px] font-bold text-slate-600"
                        onClick={() => {
                          setSelectedMailboxId(item.mailbox_id);
                          setEmailPage((prev) => ({ ...prev, page: 1 }));
                        }}
                      >
                        查看此邮箱
                      </button>
                    )}
                  </div>
                  <div className="mt-2 grid gap-1 text-xs text-slate-500">
                    {item.oauth_provider === "microsoft_graph" ? <><span>连接方式：Microsoft Graph OAuth</span><span className={item.oauth_connected ? "font-semibold text-emerald-700" : "font-semibold text-amber-700"}>授权状态：{item.oauth_connected ? "已授权" : "未授权"}</span></> : <><span>SMTP：{item.smtp_host || "-"}:{item.smtp_port || "-"}</span><span>IMAP：{item.imap_host || "-"}:{item.imap_port || "-"}</span></>}
                    <span>最近同步：{item.last_sync_at ? item.last_sync_at.replace("T", " ").slice(0, 19) : "尚未同步"}</span>
                    <span>同步状态：{item.last_sync_status || "-"}</span>
                  </div>
                  <div className="mt-3 flex items-start justify-between gap-3">
                    <span className="min-w-0 flex-1 break-words text-xs leading-5 text-slate-500">{item.last_sync_message || "尚未同步"}</span>
                    <div className="flex shrink-0 gap-2">
                      {item.oauth_provider === "microsoft_graph" ? <button className="h-9 whitespace-nowrap rounded-full bg-blue-600 px-4 text-xs font-semibold text-white" onClick={() => void connectOutlookMailbox(item.mailbox_id)}>{item.oauth_connected ? "重新授权" : "绑定 Outlook"}</button> : null}
                      <button className="h-9 whitespace-nowrap rounded-full bg-slate-900 px-4 text-xs font-semibold text-white" onClick={() => void syncMailbox(item.mailbox_id)}>
                        手动同步
                      </button>
                      <button className="grid size-9 place-items-center rounded-full text-slate-400 hover:bg-rose-50 hover:text-rose-600" title="删除邮箱配置" aria-label="删除邮箱配置" onClick={() => void deleteMailbox(item.mailbox_id)}>
                        <X size={16} />
                      </button>
                    </div>
                  </div>
                </article>
              ))}
            </div>
          </Panel>

          <Panel title="收件箱">
            <div className="mb-4 flex flex-wrap items-center justify-between gap-2 rounded-2xl bg-slate-50 px-4 py-3 text-xs text-slate-500">
              <span>当前排序：收件时间倒序，时间相同按 IMAP UID 倒序</span>
              <span>
                第 {emailPage.page} 页 / 共 {Math.max(1, Math.ceil(emailPage.total / emailPage.pageSize))} 页，共 {emailPage.total} 封
              </span>
            </div>
            {emails.length ? (
              <div className="space-y-3">
                {emails.map((item, index) => (
                  <article key={item.email_id} className="rounded-2xl border border-slate-100 p-4">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="rounded-full bg-slate-100 px-2 py-1 text-[11px] font-bold text-slate-500">#{(emailPage.page - 1) * emailPage.pageSize + index + 1}</span>
                          <div className="font-semibold text-slate-900">{item.subject}</div>
                          {item.is_read ? <span className="rounded-full bg-slate-100 px-2 py-1 text-[11px] font-bold text-slate-500">已读</span> : null}
                          {item.is_starred ? <span className="rounded-full bg-amber-100 px-2 py-1 text-[11px] font-bold text-amber-700">已星标</span> : null}
                        </div>
                        <div className="text-xs text-slate-500">来自 {item.sender.name} / {item.sender.email}</div>
                      </div>
                      <div className="text-right text-[11px] text-slate-500">
                        <div>{item.priority_label}</div>
                        <div>{item.received_at?.slice(0, 10)}</div>
                      </div>
                    </div>
                    <p className="mt-2 text-sm text-slate-600">{item.summary || item.preview}</p>
                    <div className="mt-3 flex gap-2">
                      <button
                        className={`rounded-full px-3 py-1 text-xs font-semibold ${item.is_read ? "bg-slate-200 text-slate-500" : "bg-slate-100 text-slate-700"}`}
                        disabled={item.is_read}
                        onClick={() => void markRead(item.email_id)}
                      >
                        {item.is_read ? "已读" : "标记已读"}
                      </button>
                      <button
                        className={`rounded-full px-3 py-1 text-xs font-semibold ${item.is_starred ? "bg-amber-100 text-amber-700" : "bg-slate-100 text-slate-700"}`}
                        onClick={() => void starEmail(item.email_id)}
                      >
                        {item.is_starred ? "已星标" : "星标"}
                      </button>
                      <button className="grid size-7 place-items-center rounded-full text-slate-400 hover:bg-rose-50 hover:text-rose-600" title="删除邮件" aria-label="删除邮件" onClick={() => void deleteEmail(item.email_id)}>
                        <X size={15} />
                      </button>
                    </div>
                  </article>
                ))}
              </div>
            ) : (
              <EmptyHint text="当前没有已同步邮件。" />
            )}
            <div className="mt-4 flex items-center justify-end gap-2">
              <button
                className="rounded-full bg-slate-100 px-4 py-2 text-xs font-semibold text-slate-700 disabled:cursor-not-allowed disabled:opacity-50"
                disabled={emailPage.page <= 1}
                onClick={() => setEmailPage((prev) => ({ ...prev, page: Math.max(1, prev.page - 1) }))}
              >
                上一页
              </button>
              <button
                className="rounded-full bg-slate-900 px-4 py-2 text-xs font-semibold text-white disabled:cursor-not-allowed disabled:opacity-50"
                disabled={emailPage.page >= Math.ceil(emailPage.total / emailPage.pageSize)}
                onClick={() => setEmailPage((prev) => ({ ...prev, page: prev.page + 1 }))}
              >
                下一页
              </button>
            </div>
          </Panel>
        </div>
      ) : null}

      {!isLP && activeTab === "templates" ? (
        <div className="space-y-4">
          {templateWorkspace === "draft" ? (
            <div className="template-workspace grid gap-4 lg:grid-cols-[0.95fr_1.05fr]">
              <Panel title={templateDraftMode === "create" ? "AI 对话区" : "录入对话区"} className="template-conversation-panel">
                <div className="template-mode-switch flex flex-wrap gap-2">
                  <button
                    className={`template-mode-button rounded-full px-4 py-2 text-sm font-semibold ${templateDraftMode === "create" ? "bg-slate-900 text-white" : "bg-white text-slate-600 ring-1 ring-slate-200"}`}
                    onClick={() => resetTemplateConversation("create")}
                  >
                    <Plus size={16} aria-hidden="true" /> 新建模板
                  </button>
                  <button
                    className={`template-mode-button rounded-full px-4 py-2 text-sm font-semibold ${templateDraftMode === "import" ? "bg-slate-900 text-white" : "bg-white text-slate-600 ring-1 ring-slate-200"}`}
                    onClick={() => resetTemplateConversation("import")}
                  >
                    <FileUp size={16} aria-hidden="true" /> 录入模板
                  </button>
                  <button
                  className="template-mode-button rounded-full bg-white px-4 py-2 text-sm font-semibold text-slate-700 ring-1 ring-slate-200"
                  onClick={() => setTemplateWorkspace("library")}
                >
                  <Library size={16} aria-hidden="true" /> 模板中心
                </button>
              </div>

                <div className="template-form-grid mt-5 grid gap-3 sm:grid-cols-2">
                  <Input label="模板名称" value={templateForm.name} onChange={(value) => setTemplateForm((prev) => ({ ...prev, name: value }))} />
                  <label className="grid gap-2 text-sm font-semibold text-slate-700">
                    <span>模板分类</span>
                    <select
                      className="communication-field rounded-2xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-slate-400"
                      value={templateForm.category}
                      onChange={(event) => setTemplateForm((prev) => ({ ...prev, category: event.target.value }))}
                    >
                      <option value="notice">通知</option>
                      <option value="email">邮件</option>
                      <option value="cooperation">合作函件</option>
                    </select>
                  </label>
                  <Input label="适用场景" value={templateForm.scene} onChange={(value) => setTemplateForm((prev) => ({ ...prev, scene: value }))} />
                  <Input label="使用边界" value={templateForm.usage_boundary} onChange={(value) => setTemplateForm((prev) => ({ ...prev, usage_boundary: value }))} />

                  {templateDraftMode === "import" ? (
                    <label className="template-import-zone grid gap-2 text-sm font-semibold text-slate-700">
                      <span>上传 DOCX 模板</span>
                      <input
                        className="rounded-2xl border border-slate-200 px-4 py-3 text-sm"
                        type="file"
                        accept=".doc,.docx"
                        onChange={(event) => {
                          const file = event.target.files?.[0];
                          if (!file) return;
                          void readFileAsBase64(file).then((base64) => {
                            setImportTemplateFileName(file.name);
                            setImportTemplateBase64(base64);
                          });
                        }}
                      />
                      {importTemplateFileName ? <span className="text-xs font-normal text-slate-500">当前文件：{importTemplateFileName}</span> : null}
                    </label>
                  ) : null}

                  <div className="template-chat-panel rounded-2xl border border-slate-100 bg-slate-50 p-4">
                    <div className="template-chat-panel__header"><span><Bot size={17} aria-hidden="true" /> 对话区</span><small>持续澄清</small></div>
                    <div className="template-chat-scroll mt-3 max-h-80 space-y-3 overflow-y-auto pr-1">
                      {templateChatMessages.length ? (
                        templateChatMessages.map((item) => (
                          <div key={item.id} className={`template-chat-message flex ${item.role === "user" ? "justify-end" : "justify-start"}`}>
                            <div
                              className={`max-w-[85%] rounded-2xl px-4 py-3 text-sm leading-7 ${
                                item.role === "user" ? "template-chat-bubble--user bg-slate-900 text-white" : "template-chat-bubble--assistant bg-white text-slate-700 ring-1 ring-slate-200"
                              }`}
                            >
                              <div className="mb-1 text-[11px] font-bold opacity-70">{item.role === "user" ? "你" : "AI"}</div>
                              <div className="whitespace-pre-wrap">{item.text}</div>
                            </div>
                          </div>
                        ))
                      ) : (
                        <div className="template-chat-empty rounded-xl bg-white px-4 py-3 text-sm leading-7 text-slate-600 ring-1 ring-slate-200">
                          {templateDraftMode === "create"
                            ? "请先输入你的模板需求，AI 会逐轮追问，直到形成可用草稿。"
                            : "请先上传并解析 DOCX 模板，AI 会在需要时继续追问缺失信息。"}
                        </div>
                      )}
                      {templateActionLoading === "chat" || templateActionLoading === "import" ? (
                        <div className="flex justify-start">
                          <div className="template-chat-loading rounded-2xl bg-white px-4 py-3 text-sm text-slate-500 ring-1 ring-slate-200">AI 正在处理中...</div>
                        </div>
                      ) : null}
                    </div>
                  </div>

                  <label className="template-composer grid gap-2 text-sm font-semibold text-slate-700">
                    <span>{templateDraftMode === "create" ? "发送给 AI" : "继续补充 / 修改"}</span>
                    <textarea
                      className="min-h-28 rounded-2xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-slate-400"
                      value={templateChatInput}
                      onChange={(event) => setTemplateChatInput(event.target.value)}
                      placeholder={templateDraftMode === "create" ? "例如：给我写一个年度总结报告，面向 LP，邮件正文和正式通知函都要。" : "例如：改成面向 LP，语气更正式，新增 reporting_period 字段。"}
                    />
                  </label>

                  <div className="template-send-row flex flex-wrap gap-2">
                    {templateDraftMode === "import" ? (
                      <button
                        className="template-send-button rounded-full bg-slate-900 px-4 py-2 text-sm font-semibold text-white"
                        disabled={templateActionLoading !== ""}
                        onClick={() => {
                          setTemplateActionLoading(importTemplateBase64 && !templateDraft ? "import" : "chat");
                          const action = importTemplateBase64 && !templateDraft ? draftTemplateFromImport() : draftTemplateFromChat();
                          void action.finally(() => setTemplateActionLoading(""));
                        }}
                      >
                        <Send size={16} aria-hidden="true" /> {templateActionLoading === "import"
                          ? "解析中..."
                          : templateActionLoading === "chat"
                            ? "发送中..."
                            : importTemplateBase64 && !templateDraft
                              ? "解析录入模板"
                              : "发送修改意见"}
                      </button>
                    ) : (
                      <button
                        className="template-send-button rounded-full bg-slate-900 px-4 py-2 text-sm font-semibold text-white"
                        disabled={templateActionLoading !== ""}
                        onClick={() => {
                          setTemplateActionLoading("chat");
                          void draftTemplateFromChat().finally(() => setTemplateActionLoading(""));
                        }}
                      >
                        <Send size={16} aria-hidden="true" /> {templateActionLoading === "chat" ? "发送中..." : "发送给 AI"}
                      </button>
                    )}
                  </div>
                </div>
              </Panel>

              <Panel title="草稿中心">
                {shouldRenderTemplateDraft(templateDraft, templateConfirmed, latestTemplateAssistantText) ? (
                  <div className="space-y-4 text-sm text-slate-600">
                    <AISourceBadge value={templateDraft} />
                    <div>
                      <div className="text-xs font-bold uppercase tracking-[0.12em] text-slate-400">模板摘要</div>
                      <p className="mt-2">{templateDraft.summary}</p>
                    </div>
                    <div>
                      <div className="text-xs font-bold uppercase tracking-[0.12em] text-slate-400">主题模板</div>
                      <p className="mt-2">{renderTemplatePlaceholderLabels(templateDraft.subject_template || "", usedTemplateDraftVariables)}</p>
                    </div>
                    <div>
                      <div className="text-xs font-bold uppercase tracking-[0.12em] text-slate-400">HTML 模板</div>
                      <div
                        className="mt-2 rounded-2xl bg-white p-4 ring-1 ring-slate-200"
                        dangerouslySetInnerHTML={{ __html: buildTemplateLabelPreviewHtml(templateDraft.html_template || "", usedTemplateDraftVariables) }}
                      />
                    </div>
                    <div>
                      <div className="text-xs font-bold uppercase tracking-[0.12em] text-slate-400">模板字段</div>
                      <p className="mt-2 text-xs leading-6 text-slate-500">这些字段会同时用于主题模板和 HTML 模板，正式生成函件时再由真实业务数据替换。</p>
                      <div className="mt-2 flex flex-wrap gap-2">
                        {usedTemplateDraftVariables.map((item) => (
                          <span key={item.variable_name} className="rounded-full bg-white px-3 py-1 text-xs font-semibold text-slate-700 ring-1 ring-slate-200">
                            {getVariableLabel(item)}
                          </span>
                        ))}
                      </div>
                    </div>
                    <div className="rounded-2xl bg-slate-50 p-4 ring-1 ring-slate-200">
                      <p className="text-xs leading-6 text-slate-500">
                        保存后会进入“我的私人模板”，系统内置模板仍由系统预置，普通账号不能新增或修改系统内置模板。
                      </p>
                      <button
                        className="mt-3 rounded-full bg-slate-900 px-5 py-2 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:opacity-60"
                        disabled={templateActionLoading === "save"}
                        onClick={() => {
                          setTemplateActionLoading("save");
                          void saveTemplateDraft().finally(() => setTemplateActionLoading(""));
                        }}
                      >
                        {templateActionLoading === "save" ? "保存中..." : "保存为私人模板"}
                      </button>
                    </div>
                  </div>
                ) : (
                  <EmptyHint text="AI 还在补全关键信息，形成完整草稿后会自动显示在这里。" />
                )}
              </Panel>
            </div>
          ) : (
            <Panel title="模板中心">
              <div className="template-library-hero">
                <div className="template-library-hero__copy">
                  <div className="template-library-hero__eyebrow"><Library size={15} aria-hidden="true" /> 模板资产库</div>
                  <p>统一管理系统内置模板与当前账号的私人模板。</p>
                </div>
                <div className="template-library-hero__actions">
                <button className="template-library-action template-library-action--primary" onClick={() => setTemplateWorkspace("draft")}>
                  <Plus size={16} aria-hidden="true" /> 新建模板
                </button>
                <button
                  className="template-library-action template-library-action--secondary"
                  onClick={() => {
                    setTemplateWorkspace("draft");
                    resetTemplateConversation("import");
                  }}
                >
                  <FileUp size={16} aria-hidden="true" /> 录入模板
                </button>
                </div>
              </div>
              <div className="template-library-toolbar mt-5" aria-label="模板分类筛选">
                <div className="template-library-toolbar__intro">
                  <span className="text-sm font-semibold text-slate-700">模板分类</span>
                  <span className="text-xs text-slate-400">当前展示 {libraryTemplates.length} 个模板</span>
                </div>
                <div className="template-library-filters">
                {[
                  ["all", "全部"],
                  ["notice", "通知"],
                  ["email", "邮件"],
                  ["cooperation", "合作函件"],
                ].map(([value, label]) => (
                  <button
                    key={value}
                    className={`template-library-filter ${
                      templateCategoryFilter === value
                        ? "template-library-filter--active"
                        : ""
                    }`}
                    onClick={() => setTemplateCategoryFilter(value)}
                  >
                    <span>{label}</span>
                    <small>{value === "all" ? templates.length : templates.filter((item) => item.category === value).length}</small>
                  </button>
                ))}
                </div>
              </div>
              <div className="mt-4">
                {libraryTemplates.length ? (
                  <div className="template-library-list">
                    {libraryTemplates.map((item) => (
                      <article key={item.template_id} className={`template-library-card template-library-card--${item.category || "other"}`}>
                        <div className="template-library-card__main">
                          <div className="template-library-card__heading">
                            <div>
                              <div className="template-library-card__title">{item.name}</div>
                              <div className="template-library-card__code">{item.template_code}</div>
                            </div>
                            <div className="template-library-card__badges">
                              <span className="template-library-badge template-library-badge--category">{templateCategoryLabel(item.category)}</span>
                              <span className={`template-library-badge ${item.template_source === "system_builtin" ? "template-library-badge--system" : "template-library-badge--private"}`}>
                                {templateSourceLabel(item.template_source)}
                              </span>
                              <span className="template-library-badge template-library-badge--status">{templateStatusLabel(item.status)}</span>
                            </div>
                          </div>
                          <dl className="template-library-card__details">
                            <div>
                              <dt>使用边界</dt>
                              <dd>{item.usage_boundary?.trim() || "未设置使用范围"}</dd>
                            </div>
                            <div>
                              <dt>模板字段</dt>
                              <dd className="template-library-card__fields">
                                {item.variables.length ? item.variables.map((variable) => <span key={variable.variable_name}>{getVariableLabel(variable)}</span>) : "暂无字段"}
                              </dd>
                            </div>
                          </dl>
                        </div>
                        <div className="template-library-card__actions">
                          <button className="template-version-button" onClick={() => void toggleTemplateHistory(item.template_id)}>
                            {openedTemplateId === item.template_id ? "收起版本" : "查看版本"}
                          </button>
                        </div>
                        {openedTemplateId === item.template_id ? (
                          <div className="template-library-card__versions mt-4 rounded-2xl bg-slate-50 p-4">
                            <div className="template-library-version-list">
                              {(templateVersions[item.template_id] || []).map((version) => (
                                <article key={version.version_id} className="template-library-version-card">
                                  <div className="template-library-version-card__header">
                                    <div className="template-library-version-card__number">V{version.version_no}</div>
                                    <div className="template-library-version-card__status">{templateStatusLabel(version.status)}</div>
                                  </div>
                                  <div className="template-library-version-card__docx">DOCX：{version.docx_template_name || "未配置"}</div>
                                  <div className="template-library-version-card__fields">{version.variables.map((variable) => getVariableLabel(variable)).join("、") || "暂无字段"}</div>
                                </article>
                              ))}
                            </div>
                          </div>
                        ) : null}
                      </article>
                    ))}
                  </div>
                ) : (
                  <EmptyHint text="当前没有模板数据。" />
                )}
              </div>
            </Panel>
          )}
        </div>
      ) : null}

      {activeTab === "documents" ? (
        <div className="grid gap-4 lg:grid-cols-[0.92fr_1.08fr]">
          {!isLP ? (
            <Panel title="函件 / 通知生成">
              <div className="grid gap-3">
                <label className="grid gap-2 text-sm font-semibold text-slate-700">
                  <span>选择模板</span>
                  <select className="rounded-2xl border border-slate-200 px-4 py-3" value={selectedTemplateId} onChange={(event) => setSelectedTemplateId(event.target.value)}>
                    <option value="">请选择模板</option>
                    {selectableTemplates.map((item) => (
                      <option key={item.template_id} value={item.template_id}>
                        {item.name}
                      </option>
                    ))}
                  </select>
                </label>

                <label className="grid gap-2 text-sm font-semibold text-slate-700">
                  <span>发件邮箱</span>
                  <select
                    className="rounded-2xl border border-slate-200 px-4 py-3"
                    value={documentForm.sender_mailbox_id}
                    onChange={(event) => setDocumentForm((prev) => ({ ...prev, sender_mailbox_id: Number(event.target.value) }))}
                  >
                    <option value={0}>请选择发件邮箱</option>
                    {mailboxes.filter(canUseMailboxAsSender).map((item) => (
                      <option key={item.mailbox_id} value={item.mailbox_id}>
                        {item.name} / {item.email_address}
                      </option>
                    ))}
                  </select>
                </label>

                <label className="grid gap-2 text-sm font-semibold text-slate-700">
                  <span>LP 收件对象</span>
                  <select className="rounded-2xl border border-slate-200 px-4 py-3" value={documentForm.lp_user_id} onChange={(event) => chooseLPRecipient(event.target.value)}>
                    <option value="">请选择 LP</option>
                    {lpRecipients.map((item) => (
                      <option key={item.user_id} value={item.user_id}>
                        {item.name} / {item.email}
                      </option>
                    ))}
                  </select>
                </label>

                <Input label="收件人" value={documentForm.recipient_name} onChange={(value) => setDocumentForm((prev) => ({ ...prev, recipient_name: value }))} />
                <div className="grid gap-3">
                  <div className="flex items-center justify-between gap-3">
                    <span className="text-sm font-semibold text-slate-700">收件邮箱 1</span>
                    <button
                      type="button"
                      title="添加收件邮箱"
                      aria-label="添加收件邮箱"
                      className="grid size-8 place-items-center rounded-full bg-slate-900 text-white"
                      onClick={() => setAdditionalRecipientEmails((prev) => [...prev, ""])}
                    >
                      <Plus size={16} />
                    </button>
                  </div>
                  <input
                    className="rounded-2xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-slate-400"
                    value={documentForm.recipient_email}
                    onChange={(event) => setDocumentForm((prev) => ({ ...prev, recipient_email: event.target.value }))}
                  />
                  {additionalRecipientEmails.map((email, index) => (
                    <div key={index} className="grid gap-2">
                      <div className="flex items-center justify-between gap-3">
                        <span className="text-sm font-semibold text-slate-700">收件邮箱 {index + 2}</span>
                        <button
                          type="button"
                          title={`删除收件邮箱 ${index + 2}`}
                          aria-label={`删除收件邮箱 ${index + 2}`}
                          className="grid size-8 place-items-center rounded-full text-slate-400 hover:bg-rose-50 hover:text-rose-600"
                          onClick={() => setAdditionalRecipientEmails((prev) => prev.filter((_, itemIndex) => itemIndex !== index))}
                        >
                          <X size={16} />
                        </button>
                      </div>
                      <input
                        className="rounded-2xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-slate-400"
                        value={email}
                        onChange={(event) => setAdditionalRecipientEmails((prev) => prev.map((item, itemIndex) => (itemIndex === index ? event.target.value : item)))}
                      />
                    </div>
                  ))}
                </div>
                <Input label="标题" value={documentForm.title} onChange={(value) => setDocumentForm((prev) => ({ ...prev, title: value }))} />
                <Input label="基金名称" value={documentForm.fund_name} onChange={(value) => setDocumentForm((prev) => ({ ...prev, fund_name: value }))} />
                <Input label="收件对象名称" value={documentForm.lp_name} onChange={(value) => setDocumentForm((prev) => ({ ...prev, lp_name: value }))} />
                <Input label="通知日期" value={documentForm.notice_date} onChange={(value) => setDocumentForm((prev) => ({ ...prev, notice_date: value }))} />

                {extraDocumentVariables.map((item) => (
                  <Input
                    key={item.variable_name}
                    label={getVariableLabel(item)}
                    value={deriveDocumentVariableValue(item.variable_name, documentForm, documentVariableValues)}
                    onChange={(value) => setDocumentVariableValues((prev) => ({ ...prev, [item.variable_name]: value }))}
                  />
                ))}

                <div className="flex flex-wrap gap-2">
                  <button
                    className="rounded-full bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-700"
                    disabled={actionLoading !== ""}
                    onClick={() => void runDocumentAction("assist", "正在生成正文建议...", askRenderAssist)}
                  >
                    {actionLoading === "assist" ? "正文建议生成中..." : "AI 生成正文建议"}
                  </button>
                  <button
                    className="rounded-full bg-slate-900 px-4 py-2 text-sm font-semibold text-white"
                    disabled={actionLoading !== ""}
                    onClick={() => {
                      void runDocumentAction("create", "正在创建函件...", createDocument);
                    }}
                  >
                    {actionLoading === "create" ? "函件创建中..." : "创建函件"}
                  </button>
                </div>
                {message ? (
                  <div className="rounded-2xl bg-slate-50 px-4 py-3 text-sm font-semibold text-slate-600">
                    {message}
                  </div>
                ) : null}
              </div>

              {renderAssist ? (
                <DocumentRenderAssistPanel
                  value={renderAssist}
                  variables={buildDocumentVariables()}
                  draftTitle={documentDraftTitle}
                  draftHtml={documentDraftHtml}
                  onChangeTitle={setDocumentDraftTitle}
                  onChangeHtml={setDocumentDraftHtml}
                />
              ) : null}
            </Panel>
          ) : null}

          <Panel title={isLP ? "我的通知函件" : "函件实例"}>
            {documents.length ? (
              <div className="space-y-3">
                {documents.map((item) => {
                  const hasFormalDocx = Boolean(item.docx_file_name);
                  return (
                    <article key={item.document_id} className="rounded-2xl border border-slate-100 p-4">
                      <div className="flex flex-col gap-3 sm:flex-row sm:items-stretch">
                        <div className="min-w-0 flex-1">
                          <div className="font-semibold text-slate-900">{item.title}</div>
                          <div className="text-xs text-slate-500">
                            {documentStatusLabel(item.status)} / {item.recipient_email}
                          </div>
                          {item.sender_mailbox_email ? <div className="mt-1 text-xs text-slate-500">发送邮箱：{item.sender_mailbox_name || "-"} / {item.sender_mailbox_email}</div> : null}
                          {item.cc_recipients?.length ? <div className="mt-1 text-xs text-slate-500">其他收件邮箱：{item.cc_recipients.map((recipient) => recipient.email || recipient.name).join("、")}</div> : null}
                          {item.sent_at ? <div className="mt-1 text-xs text-slate-500">发送时间：{item.sent_at}</div> : null}
                          {item.send_result ? <div className="mt-1 text-xs text-slate-500">发送结果：{sendResultLabel(item.send_result)}</div> : null}
                        </div>
                        <div className="flex shrink-0 flex-wrap justify-end gap-2 self-end sm:self-auto sm:items-end">
                          {!isLP && canSubmitDocument(item.status) ? (
                            <button className="whitespace-nowrap rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-700" onClick={() => void submitDocument(item.document_id)}>
                              提审
                            </button>
                          ) : null}
                          {!isLP && canSendDocument(item.status, hasFormalDocx, item.submitted_by_user_id, user?.user_id) ? (
                            <button className="whitespace-nowrap rounded-full bg-slate-900 px-3 py-1 text-xs font-semibold text-white" onClick={() => void sendDocument(item.document_id)}>
                              发送                            </button>
                          ) : null}
                          {canDownloadDocument(item.status, hasFormalDocx) ? (
                            <button className="whitespace-nowrap rounded-full bg-white px-3 py-1 text-xs font-semibold text-slate-700 ring-1 ring-slate-200" onClick={() => void downloadDocument(item)}>
                              下载 DOCX
                            </button>
                          ) : null}
                        </div>
                      </div>
                    </article>
                  );
                })}
              </div>
            ) : (
              <EmptyHint text="当前还没有函件实例。" />
            )}
          </Panel>
        </div>
      ) : null}
    </section>
  );
}

function DocumentRenderAssistPanel({
  value,
  variables,
  draftTitle,
  draftHtml,
  onChangeTitle,
  onChangeHtml,
}: {
  value: TemplateAssistResponse;
  variables: Record<string, unknown>;
  draftTitle: string;
  draftHtml: string;
  onChangeTitle: (value: string) => void;
  onChangeHtml: (value: string) => void;
}) {
  const previewHtml = React.useMemo(() => buildClientPreviewHtml(draftHtml || value.html_template || "", variables), [draftHtml, value.html_template, variables]);
  return (
    <div id="communication-render-assist" className="mt-5 rounded-2xl border border-emerald-100 bg-emerald-50/70 p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="text-sm font-black text-slate-900">AI 正文建议</div>
        <AISourceBadge value={value} compact />
      </div>
      <p className="mt-2 text-sm text-slate-600">{value.summary}</p>
      {value.subject_template ? (
        <div className="mt-4 rounded-xl bg-white p-3 ring-1 ring-emerald-100">
          <div className="text-xs font-bold uppercase tracking-[0.12em] text-slate-400">建议标题</div>
          <div className="mt-2 text-sm font-semibold text-slate-900">{value.subject_template}</div>
        </div>
      ) : null}
      {value.suggested_fields.length ? (
        <div className="mt-4 rounded-xl bg-white p-3 ring-1 ring-emerald-100">
          <div className="text-xs font-bold uppercase tracking-[0.12em] text-slate-400">建议字段</div>
          <div className="mt-2 flex flex-wrap gap-2">
            {value.suggested_fields.map((item) => (
              <span key={item.variable_name} className="rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-700">
                {getVariableLabel(item)}
              </span>
            ))}
          </div>
        </div>
      ) : null}
      {previewHtml ? (
        <div className="mt-4 rounded-xl bg-white p-4 ring-1 ring-emerald-100">
          <div className="text-xs font-bold uppercase tracking-[0.12em] text-slate-400">正文建议预览</div>
          <div className="prose prose-sm mt-3 max-w-none text-slate-700" dangerouslySetInnerHTML={{ __html: previewHtml }} />
        </div>
      ) : null}
      <div className="mt-4 rounded-xl bg-white p-4 ring-1 ring-emerald-100">
        <div className="text-xs font-bold uppercase tracking-[0.12em] text-slate-400">人工确认后编辑</div>
        <p className="mt-2 text-xs leading-6 text-slate-500">这里可以手动修改标题和正文。创建函件和后续发送会使用这里的内容。</p>
        <label className="mt-4 grid gap-2 text-sm font-semibold text-slate-700">
          <span>可编辑标题</span>
          <input
            className="rounded-2xl border border-emerald-200 px-4 py-3 text-sm outline-none focus:border-emerald-400"
            value={draftTitle}
            onChange={(event) => onChangeTitle(event.target.value)}
            placeholder="请确认或修改邮件标题"
          />
        </label>
        <label className="mt-4 grid gap-2 text-sm font-semibold text-slate-700">
          <span>可编辑正文</span>
          <RichTextEditor
            value={draftHtml}
            onChange={onChangeHtml}
          />
        </label>
      </div>
      {value.follow_up_questions.length ? (
        <div className="mt-4">
          <div className="text-xs font-bold uppercase tracking-[0.12em] text-slate-400">待补充问题</div>
          <div className="mt-2 space-y-2">
            {value.follow_up_questions.map((item) => (
              <div key={item} className="rounded-xl bg-white px-3 py-2 text-sm text-slate-600 ring-1 ring-emerald-100">
                {item}
              </div>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
}


function AISourceBadge({ value, compact = false }: { value: TemplateAssistResponse; compact?: boolean }) {
  const isFallback = value.ai_source === "local_fallback";
  const label = isFallback ? "本地兜底" : "真实 AI";
  const message = value.ai_source_message || (isFallback ? "Python AI 服务不可用，当前为本地兜底结果。" : "已连接 Python AI 服务。");
  const className = isFallback
    ? "bg-amber-50 text-amber-700 ring-amber-200"
    : "bg-emerald-50 text-emerald-700 ring-emerald-200";

  return (
    <div className={`rounded-2xl px-3 py-2 text-xs font-semibold ring-1 ${className} ${compact ? "" : "inline-flex"}`} title={message}>
      {label}
      {!compact ? <span className="ml-2 font-normal">{message}</span> : null}
    </div>
  );
}

function RichTextEditor({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  const editorRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    const editor = editorRef.current;
    if (editor && editor.innerHTML !== value) {
      editor.innerHTML = value;
    }
  }, [value]);

  return (
    <div
      ref={editorRef}
      contentEditable
      suppressContentEditableWarning
      role="textbox"
      aria-multiline="true"
      className="prose prose-sm min-h-52 max-w-none rounded-2xl border border-emerald-200 bg-white px-4 py-3 leading-7 text-slate-700 outline-none focus:border-emerald-400"
      onInput={(event) => onChange(event.currentTarget.innerHTML)}
    />
  );
}

function MetricCard({ label, value, hint }: { label: string; value: string; hint: string }) {
  const metricVisuals: Record<string, { tone: string; Icon: typeof Mail }> = {
    邮箱数量: { tone: "mail", Icon: Mail },
    模板数量: { tone: "template", Icon: FileText },
    函件数量: { tone: "document", Icon: Send },
    "未读/待回": { tone: "followup", Icon: MessageSquare },
  };
  const visual = metricVisuals[label] || metricVisuals.邮箱数量;
  const MetricIcon = visual.Icon;

  return (
    <div className={`communication-metric communication-metric--${visual.tone} rounded-[24px] bg-white p-5 shadow-sm ring-1 ring-black/5`}>
      <div className="communication-metric__header">
        <div className="text-xs font-bold uppercase tracking-[0.12em] text-slate-400">{label}</div>
        <span className="communication-metric__icon" aria-hidden="true"><MetricIcon size={18} strokeWidth={1.9} /></span>
      </div>
      <div className="communication-metric__value mt-3 text-3xl font-black text-slate-900">{value}</div>
      <div className="communication-metric__hint mt-2 text-sm text-slate-500">{hint}</div>
      <div className="communication-metric__trace" aria-hidden="true"><span /><span /><span /><span /></div>
    </div>
  );
}

function SummaryBadge({ label, value, tone }: { label: string; value: string; tone: "amber" | "blue" | "emerald" }) {
  const toneClassMap: Record<typeof tone, string> = {
    amber: "bg-amber-50 text-amber-700 ring-amber-100",
    blue: "bg-sky-50 text-sky-700 ring-sky-100",
    emerald: "bg-emerald-50 text-emerald-700 ring-emerald-100",
  };

  return (
    <div className={`communication-summary-badge communication-summary-badge--${tone} rounded-2xl px-4 py-3 ring-1 ${toneClassMap[tone]}`}>
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2 text-xs font-bold tracking-[0.12em]">
          <span className="communication-status-light" aria-hidden="true" />
          {label}
        </div>
        <span className="communication-live-label">实时</span>
      </div>
      <div className="mt-2 text-2xl font-black">{value}</div>
      <div className="mt-1 text-xs font-medium opacity-70">收件箱状态</div>
    </div>
  );
}

function Panel({ title, children, className = "" }: { title: string; children: React.ReactNode; className?: string }) {
  return (
    <section className={`communication-panel rounded-[24px] bg-white p-6 shadow-sm ring-1 ring-black/5 ${className}`}>
      <h2 className="text-lg font-black text-slate-900">{title}</h2>
      <div className="mt-4">{children}</div>
    </section>
  );
}

function EmptyHint({ text }: { text: string }) {
  return <div className="communication-empty rounded-xl bg-slate-50 px-4 py-3 text-sm text-slate-500">{text}</div>;
}

function Input({ label, value, onChange, type = "text" }: { label: string; value: string | number; onChange: (value: string) => void; type?: string }) {
  return (
    <label className="grid gap-2 text-sm font-semibold text-slate-700">
      <span>{label}</span>
      <input className="communication-field rounded-2xl border border-slate-200 px-4 py-3 text-sm outline-none focus:border-slate-400" type={type} value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="communication-info-row flex items-center justify-between gap-3 rounded-xl bg-slate-50 px-4 py-3">
      <span className="text-slate-500">{label}</span>
      <span className="font-semibold text-slate-900">{value}</span>
    </div>
  );
}

export default CommunicationCenterView;



