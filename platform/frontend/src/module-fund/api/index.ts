import type {
  ComplianceCheckRequest,
  CreateFundRequest,
  CreateLPRequest,
  Dict,
  FundItem,
  FundQuery,
  LPItem,
  LPQuery,
  PageResult,
} from "@/base/api";
import { request } from "@/base/api";

// 本文件封装募资管理 FUND-* 接口。
export const fundApi = {
  listLPs(query?: LPQuery) {
    return request.get<PageResult<LPItem>>("/fundraising/lps", query);
  },
  createLP(payload: CreateLPRequest) {
    return request.post<LPItem>("/fundraising/lps", payload);
  },
  matchLP(payload: Dict) {
    return request.post<Dict[]>("/fundraising/lps/match", payload);
  },
  getLPProfile(lpId: string) {
    return request.get<Dict>(`/fundraising/lps/${lpId}/profile`);
  },
  listFunds(query?: FundQuery) {
    return request.get<PageResult<FundItem>>("/fundraising/funds", query);
  },
  createFund(payload: CreateFundRequest) {
    return request.post<FundItem>("/fundraising/funds", payload);
  },
  subscribeFund(fundId: string, payload: Dict) {
    return request.post<Dict>(`/fundraising/funds/${fundId}/subscribe`, payload);
  },
  distributeFund(fundId: string, payload: Dict) {
    return request.post<Dict>(`/fundraising/funds/${fundId}/distributions`, payload);
  },
  complianceCheck(payload: ComplianceCheckRequest) {
    return request.post<Dict>("/fundraising/compliance/check", payload);
  },
  generateComplianceReport(fundId: string, payload: Dict) {
    return request.post<Dict>(`/fundraising/funds/${fundId}/compliance-report`, payload);
  },
  listComplianceHistory(fundId: string) {
    return request.get<Dict[]>(`/fundraising/funds/${fundId}/compliance-history`);
  },
};
