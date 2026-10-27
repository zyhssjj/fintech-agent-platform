import { request } from "@/base/api";
import { authStore } from "@/base/stores/authStore";

export interface Pagination {
  page: number;
  page_size: number;
  total: number;
}

export interface PageResult<T> {
  list: T[];
  pagination: Pagination;
}

export interface PersonRef {
  user_id?: string;
  name: string;
  email?: string;
}

export interface MailboxAccount {
  mailbox_id: number;
  mailbox_code: string;
  name: string;
  mailbox_type: string;
  email_address: string;
  auth_type: string;
  smtp_host: string;
  smtp_port: number;
  imap_host: string;
  imap_port: number;
  username: string;
  oauth_provider: string;
  signature_html: string;
  is_default_sender: boolean;
  status: string;
  last_sync_at?: string;
  last_sync_status?: string;
  last_sync_message?: string;
  last_imap_uid?: number;
  last_imap_uid_validity?: number;
  oauth_connected?: boolean;
}

export interface EmailItem {
  email_id: string;
  mailbox_id: number;
  folder: string;
  subject: string;
  preview: string;
  body_html?: string;
  sender: PersonRef;
  recipients: PersonRef[];
  cc_recipients: PersonRef[];
  bcc_recipients: PersonRef[];
  summary: string;
  reply_draft: string;
  priority_label: string;
  priority_score: number;
  needs_reply: boolean;
  is_read: boolean;
  is_starred: boolean;
  has_attachments: boolean;
  status: string;
  received_at?: string;
  sent_at?: string;
}

export interface DailySummary {
  date: string;
  total: number;
  unread: number;
  needs_reply: number;
  top_priority: EmailItem[];
  daily_brief: string;
}

export interface TemplateVariable {
  variable_name: string;
  variable_label: string;
  variable_type: string;
  is_required: boolean;
  default_value: string;
}

export interface TemplateVersion {
  version_id: string;
  version_no: number;
  subject_template: string;
  html_content: string;
  docx_template_name: string;
  docx_template_base64?: string;
  status: string;
  approval_id: string;
  submitted_at?: string;
  published_at?: string;
  variables: TemplateVariable[];
}

export interface TemplateItem {
  template_id: string;
  template_code: string;
  name: string;
  category: string;
  scene: string;
  usage_boundary?: string;
  template_source?: string;
  owner_user_id?: string;
  source_ref_id?: string;
  is_editable?: boolean;
  ai_session_id?: string;
  import_file_name?: string;
  parse_status?: string;
  status: string;
  published_version_id?: string;
  current_version?: TemplateVersion;
  variables: TemplateVariable[];
}

export interface TemplateAssistResponse {
  summary: string;
  follow_up_questions: string[];
  suggested_fields: TemplateVariable[];
  subject_template: string;
  html_template: string;
  ai_source?: "ai_service" | "local_fallback" | string;
  ai_source_message?: string;
  plain_text?: string;
  paragraphs?: string[];
}

export interface DocumentItem {
  document_id: string;
  template_id: string;
  title: string;
  document_type: string;
  recipient_name: string;
  recipient_email: string;
  cc_recipients: PersonRef[];
  sender_mailbox_id?: number;
  sender_mailbox_name?: string;
  sender_mailbox_email?: string;
  html_content: string;
  docx_file_name: string;
  docx_base64?: string;
  status: string;
  approval_id: string;
  submitted_by_user_id: string;
  variables: Record<string, unknown>;
  send_result?: string;
  sent_at?: string;
}

export interface LPRecipient {
  user_id: string;
  name: string;
  email: string;
  username: string;
}

export interface ApprovalItem {
  approval_id: string;
  business_type: string;
  business_id: string;
  title: string;
  status: string;
  current_step_no: number;
  current_approver_role: string;
  applicant_id: string;
  applicant_name: string;
  deadline_at?: string;
  finished_at?: string;
  is_overdue: boolean;
  payload: Record<string, unknown>;
  progress_log: Array<Record<string, unknown>>;
}

export interface MailboxPayload {
  name: string;
  mailbox_type: string;
  email_address: string;
  auth_type: string;
  smtp_host: string;
  smtp_port: number;
  imap_host: string;
  imap_port: number;
  username: string;
  password: string;
  oauth_provider: string;
  signature_html: string;
  is_default_sender: boolean;
}

export interface TemplatePayload {
  template_code?: string;
  name: string;
  category: string;
  scene: string;
  usage_boundary?: string;
  ai_session_id?: string;
  import_file_name?: string;
  parse_status?: string;
  subject_template: string;
  html_content: string;
  docx_template_name?: string;
  docx_template_base64?: string;
  variables: TemplateVariable[];
}

export interface TemplateChatAssistPayload {
  scene?: string;
  goal: string;
  current_summary?: string;
  latest_user_message: string;
  known_fields?: TemplateVariable[];
}

export interface TemplateImportAssistPayload {
  file_name: string;
  scene?: string;
  docx_template_base64: string;
}

export interface TemplateRenderAssistPayload {
  template_name?: string;
  template_scene?: string;
  recipient_type?: string;
  business_goal: string;
  known_values?: Record<string, unknown>;
}

export interface DocumentRenderPayload {
  template_id: string;
  recipient_name: string;
  recipient_email: string;
  cc_recipients: PersonRef[];
  sender_mailbox_id: number;
  title: string;
  html_content?: string;
  variables: Record<string, unknown>;
}

export interface DocumentCreatePayload extends DocumentRenderPayload {
  visible_user_ids: string[];
  document_type: string;
}

async function fetchBlob(path: string, fileName: string) {
  const token = authStore.getSnapshot().accessToken;
  const baseURL = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim() || "/v1";
  const resp = await fetch(`${baseURL}${path}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!resp.ok) throw new Error(`下载失败: ${resp.status}`);
  const blob = await resp.blob();
  const url = window.URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName;
  anchor.click();
  window.URL.revokeObjectURL(url);
}

export const communicationApi = {
  listMailboxes: () => request.get<MailboxAccount[]>("/agents/communication/mailboxes"),
  createMailbox: (payload: MailboxPayload) => request.post<MailboxAccount>("/agents/communication/mailboxes", payload),
  deleteMailbox: (id: number) => request.delete<Record<string, unknown>>(`/agents/communication/mailboxes/${id}`),
  beginOutlookAuthorization: (id: number) => request.post<{ authorization_url: string }>(`/agents/communication/mailboxes/${id}/outlook/authorize`),
  syncMailbox: (id: number) => request.post<Record<string, unknown>>(`/agents/communication/mailboxes/${id}/sync`),
  listEmails: (query?: Record<string, unknown>) => request.get<PageResult<EmailItem>>("/agents/communication/emails", query),
  emailSummary: (mailboxId?: number) => request.get<DailySummary>("/agents/communication/emails/summary", mailboxId ? { mailbox_id: mailboxId } : undefined),
  markRead: (id: string) => request.post<Record<string, unknown>>(`/agents/communication/emails/${id}/read`),
  starEmail: (id: string) => request.post<{ status: string; is_starred: boolean }>(`/agents/communication/emails/${id}/star`),
  deleteEmail: (id: string) => request.delete<Record<string, unknown>>(`/agents/communication/emails/${id}`),
  listTemplates: (query?: Record<string, unknown>) => request.get<PageResult<TemplateItem>>("/agents/communication/templates", query),
  draftTemplateFromChat: (payload: TemplateChatAssistPayload) => request.post<TemplateAssistResponse>("/agents/communication/templates/ai/chat", payload),
  draftTemplateFromImport: (payload: TemplateImportAssistPayload) => request.post<TemplateAssistResponse>("/agents/communication/templates/ai/import", payload),
  assistTemplateRender: (payload: TemplateRenderAssistPayload) => request.post<TemplateAssistResponse>("/agents/communication/templates/ai/render-assist", payload),
  createTemplate: (payload: TemplatePayload) => request.post<TemplateItem>("/agents/communication/templates", payload),
  listTemplateVersions: (id: string) => request.get<TemplateVersion[]>(`/agents/communication/templates/${id}/versions`),
  submitTemplate: (id: string) => request.post<Record<string, unknown>>(`/agents/communication/templates/${id}/submit`),
  renderDocument: (payload: DocumentRenderPayload) => request.post<DocumentItem>("/agents/communication/documents/render", payload),
  createDocument: (payload: DocumentCreatePayload) => request.post<DocumentItem>("/agents/communication/documents", payload),
  listDocuments: (query?: Record<string, unknown>) => request.get<PageResult<DocumentItem>>("/agents/communication/documents", query),
  listLPRecipients: () => request.get<LPRecipient[]>("/agents/communication/recipients/lps"),
  submitDocument: (id: string) => request.post<Record<string, unknown>>(`/agents/communication/documents/${id}/submit`),
  sendDocument: (id: string) => request.post<Record<string, unknown>>(`/agents/communication/documents/${id}/send`),
  downloadDocument: (id: string, fileName: string) => fetchBlob(`/agents/communication/documents/${id}/download`, fileName),
  listApprovals: (query?: Record<string, unknown>) => request.get<PageResult<ApprovalItem>>("/approvals", query),
  actApproval: (id: string, action: string, comment: string) => request.post<ApprovalItem>(`/approvals/${id}/action`, { action, comment }),
};
