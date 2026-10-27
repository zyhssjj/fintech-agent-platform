import { useApiResource } from "@/base/composables/useApiResource";
import { fundApi } from "@/module-fund/api";

// 本文件提供 LP 列表的 compose/hook。
export function useLPProfiles() {
  return useApiResource(() => fundApi.listLPs({ page: 1, page_size: 20 }), []);
}
