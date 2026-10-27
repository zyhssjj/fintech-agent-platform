# Frontend Base

This directory is the shared frontend base. Business modules should reuse it instead of rebuilding login, permission, layout, API, table, upload, modal, or notification logic.

## What Is Provided

- `api/`: shared `request` client with token injection, 401 refresh, response unwrap, global loading, and error notices.
- `router/`: route registry, role/permission checks, and authenticated route guards.
- `stores/`: Zustand stores for auth, permissions, navigation tabs, UI feedback, theme, and global app config.
- `layout/`: application shell for sidebar, topbar, tab view, and content area.
- `components/`: reusable base table, file upload, modal, tab view, and global feedback.
- `views/`: login, register, forgot password, 403, 404, 500, profile, audit log, notification center, and system config pages.

## How Business Modules Plug In

1. Create pages under `src/module-xxx/views`.
2. Put request wrappers under `src/module-xxx/api`.
3. Add module hooks under `src/module-xxx/composables`.
4. Register the menu and route in `src/base/router/moduleRegistry.ts`.
5. If a route needs custom roles or permissions, set `requiredRoles` or `requiredPermissions` in the menu record.

Module API files should import from `@/base/api`:

```ts
import { request } from "@/base/api";

export const projectApi = {
  list() {
    return request.get("/projects");
  },
};
```
