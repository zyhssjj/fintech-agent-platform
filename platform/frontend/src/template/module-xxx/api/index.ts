import { request } from "@/base/api";

export const moduleXxxApi = {
  list() {
    return request.get("/module-xxx");
  },
};
