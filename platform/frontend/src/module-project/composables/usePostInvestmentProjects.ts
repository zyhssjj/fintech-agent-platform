import { useApiResource } from "@/base/composables/useApiResource";
import { projectApi } from "@/module-project/api";

// 本文件提供投后项目列表的 compose/hook。
export function usePostInvestmentProjects() {
  return useApiResource(() => projectApi.listPostInvestmentProjects({ page: 1, page_size: 20 }), []);
}
