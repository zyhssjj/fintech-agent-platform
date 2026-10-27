import { useApiResource } from "@/base/composables/useApiResource";
import { agentApi } from "@/module-agent/api";

// 本文件提供智能体待办列表的 compose/hook。
export function useAgentTodos() {
  return useApiResource(() => agentApi.listTodos({ page: 1, page_size: 20 }), []);
}
