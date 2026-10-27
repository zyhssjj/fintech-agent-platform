import type {
  AgentDetail,
  AgentInvokeRequest,
  AgentInvokeResponse,
  AgentListQuery,
  AgentSummary,
  CreateTodoRequest,
  Dict,
  EmailItem,
  EmailQuery,
  KnowledgeSearchQuery,
  KnowledgeSearchResult,
  MeetingMinutesRequest,
  MeetingMinutesResponse,
  NotificationRequest,
  PageResult,
  SendEmailRequest,
  TodoItem,
  TodoQuery,
  UpdateTodoStatusRequest,
} from "@/base/api";
import { request } from "@/base/api";

// 本文件封装智能体模块 AGENT-* 接口。
export const agentApi = {
  listAgents(query?: AgentListQuery) {
    return request.get<PageResult<AgentSummary>>("/agents", query);
  },
  getAgent(agentId: string) {
    return request.get<AgentDetail>(`/agents/${agentId}`);
  },
  invokeAgent(agentId: string, payload: AgentInvokeRequest) {
    return request.post<AgentInvokeResponse>(`/agents/${agentId}/invoke`, payload);
  },
  listTodos(query?: TodoQuery) {
    return request.get<PageResult<TodoItem>>("/agents/customer/todos", query);
  },
  createTodo(payload: CreateTodoRequest) {
    return request.post<TodoItem>("/agents/customer/todos", payload);
  },
  updateTodoStatus(todoId: string, payload: UpdateTodoStatusRequest) {
    return request.patch<{ status: string }>(`/agents/customer/todos/${todoId}/status`, payload);
  },
  sendNotification(payload: NotificationRequest) {
    return request.post<{ status: string }>("/agents/customer/notifications", payload);
  },
  uploadKnowledgeDocument(formData: FormData) {
    return request.upload<Dict>("/agents/knowledge/documents", formData);
  },
  searchKnowledge(query: KnowledgeSearchQuery) {
    return request.post<KnowledgeSearchResult>("/agents/knowledge/search", query);
  },
  answerKnowledge(query: KnowledgeSearchQuery) {
    return request.post<KnowledgeSearchResult>("/agents/knowledge/answer", query);
  },
  getCompetitorIntelligence(query?: Dict<string | number | boolean | string[]>) {
    return request.get<Dict[]>("/agents/knowledge/competitor-intelligence", query);
  },
  listEmails(query?: EmailQuery) {
    return request.get<PageResult<EmailItem>>("/agents/communication/emails", query);
  },
  sendEmail(payload: SendEmailRequest) {
    return request.post<{ status: string }>("/agents/communication/emails", payload);
  },
  listEmailTemplates(query?: Dict<string | number | boolean>) {
    return request.get<PageResult<Dict>>("/agents/communication/templates", query);
  },
  generateMeetingMinutes(payload: MeetingMinutesRequest) {
    return request.post<MeetingMinutesResponse>("/agents/cooperation/meeting-minutes", payload);
  },
  listFileVersions(fileId: string) {
    return request.get<Dict[]>(`/agents/cooperation/files/${fileId}/versions`);
  },
};
