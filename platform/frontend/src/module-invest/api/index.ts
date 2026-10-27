import type {
  CreateInvestmentProjectRequest,
  DCFValuationRequest,
  DCFValuationResponse,
  Dict,
  InvestmentProjectItem,
  InvestmentProjectQuery,
  PageResult,
} from "@/base/api";
import { request } from "@/base/api";

// 本文件封装投资管理 INVEST-* 接口。
export const investmentApi = {
  listProjects(query?: InvestmentProjectQuery) {
    return request.get<PageResult<InvestmentProjectItem>>("/investment/projects", query);
  },
  createProject(payload: CreateInvestmentProjectRequest) {
    return request.post<InvestmentProjectItem>("/investment/projects", payload);
  },
  filterProjects(payload: Dict) {
    return request.post<PageResult<InvestmentProjectItem>>("/investment/projects/filter", payload);
  },
  createDueDiligence(projectId: string, payload: Dict) {
    return request.post<Dict>(`/investment/projects/${projectId}/due-diligence`, payload);
  },
  updateDueDiligenceProgress(ddId: string, payload: Dict) {
    return request.patch<Dict>(`/investment/due-diligence/${ddId}/progress`, payload);
  },
  completeDueDiligence(ddId: string, payload: Dict) {
    return request.post<Dict>(`/investment/due-diligence/${ddId}/complete`, payload);
  },
  createDecision(payload: Dict) {
    return request.post<Dict>("/investment/decisions", payload);
  },
  voteDecision(decisionId: string, payload: Dict) {
    return request.post<Dict>(`/investment/decisions/${decisionId}/vote`, payload);
  },
  valuateDCF(payload: DCFValuationRequest) {
    return request.post<DCFValuationResponse>("/investment/valuation/dcf", payload);
  },
  createExecution(payload: Dict) {
    return request.post<Dict>("/investment/executions", payload);
  },
  generateTerms(projectId: string, payload: Dict) {
    return request.post<Dict>(`/investment/projects/${projectId}/terms`, payload);
  },
};
