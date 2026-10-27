import type { Dict, InvestmentProjectItem, PageResult } from "@/base/api";
import { request } from "@/base/api";

// 本文件封装投后项目管理 PROJECT-* 接口。
export const projectApi = {
  listPostInvestmentProjects(query?: Dict<string | number | boolean>) {
    return request.get<PageResult<InvestmentProjectItem>>("/projects/post-investment", query);
  },
  updateOperations(projectId: string, payload: Dict) {
    return request.post<Dict>(`/projects/${projectId}/operations`, payload);
  },
  listMilestones(projectId: string) {
    return request.get<Dict[]>(`/projects/${projectId}/milestones`);
  },
  generateReport(projectId: string, payload: Dict) {
    return request.post<Dict>(`/projects/${projectId}/reports`, payload);
  },
  listReports(projectId: string) {
    return request.get<Dict[]>(`/projects/${projectId}/reports`);
  },
  getBoard(projectId: string) {
    return request.get<Dict>(`/projects/${projectId}/board`);
  },
  voteBoard(projectId: string, payload: Dict) {
    return request.post<Dict>(`/projects/${projectId}/board/votes`, payload);
  },
  listRiskAlerts(projectId: string, query?: Dict<string | number | boolean>) {
    return request.get<PageResult<Dict>>(`/projects/${projectId}/risk-alerts`, query);
  },
  resolveRiskAlert(projectId: string, alertId: string, payload: Dict) {
    return request.patch<Dict>(`/projects/${projectId}/risk-alerts/${alertId}`, payload);
  },
};
