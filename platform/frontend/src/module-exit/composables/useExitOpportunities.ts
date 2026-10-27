import { useApiResource } from "@/base/composables/useApiResource";
import { exitApi } from "@/module-exit/api";

// 本文件提供退出机会列表的 compose/hook。
export function useExitOpportunities(projectId?: string) {
  return useApiResource(() => exitApi.listOpportunities({ project_id: projectId, page: 1, page_size: 20 }), [projectId]);
}
