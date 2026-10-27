import { createElement } from "react";
import type { RouteObject } from "react-router-dom";
import { ListView } from "@/template/module-xxx/views/ListView";

export const moduleXxxRoutes: RouteObject[] = [
  {
    path: "module-xxx",
    element: createElement(ListView),
  },
];
