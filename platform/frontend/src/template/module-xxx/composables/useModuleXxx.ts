import { useApiResource } from "@/base/composables/useApiResource";
import { moduleXxxApi } from "@/template/module-xxx/api";

// 本文件是业务模块 compose/hook 模板，用于隔离页面状态和请求逻辑。
export function useModuleXxx() {
  return useApiResource(() => moduleXxxApi.list(), []);
}
