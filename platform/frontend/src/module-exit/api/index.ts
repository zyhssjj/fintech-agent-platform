import type { Dict, ExitAnalysisRequest, ExitOpportunityItem, ExitOpportunityQuery, PageResult } from "@/base/api";
import { request } from "@/base/api";

// 本文件封装退出管理 EXIT-* 接口。
export const exitApi = {
  listOpportunities(query?: ExitOpportunityQuery) {
    return request.get<PageResult<ExitOpportunityItem>>("/exit/opportunities", query);
  },
  analyzePath(projectId: string, payload: ExitAnalysisRequest) {
    return request.post<Dict>(`/exit/projects/${projectId}/analysis`, payload);
  },
  createExecution(payload: Dict) {
    return request.post<Dict>("/exit/executions", payload);
  },
  updateExecutionProgress(executionId: string, payload: Dict) {
    return request.patch<Dict>(`/exit/executions/${executionId}/progress`, payload);
  },
  calculateDistribution(executionId: string, payload: Dict) {
    return request.post<Dict>(`/exit/executions/${executionId}/distribution`, payload);
  },
  listLPDisclosures(query?: Dict<string | number | boolean>) {
    return request.get<PageResult<Dict>>("/exit/lp-disclosures", query);
  },
  generateLPDisclosures(payload: Dict) {
    return request.post<Dict>("/exit/lp-disclosures/generate", payload);
  },
  listFundMeetings(query?: Dict<string | number | boolean>) {
    return request.get<PageResult<Dict>>("/exit/fund-meetings", query);
  },
  createFundMeeting(payload: Dict) {
    return request.post<Dict>("/exit/fund-meetings", payload);
  },
  voteFundMeeting(meetingId: string, payload: Dict) {
    return request.post<Dict>(`/exit/fund-meetings/${meetingId}/vote`, payload);
  },
};
