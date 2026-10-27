import { useApiResource } from "@/base/composables/useApiResource";
import { investmentApi } from "@/module-invest/api";

// 本文件提供投资项目列表的 compose/hook。
export function useInvestmentProjects(stage?: string) {
  return useApiResource(() => investmentApi.listProjects({ stage, page: 1, page_size: 20 }), [stage]);
}
