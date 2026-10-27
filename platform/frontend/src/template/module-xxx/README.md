# Module XXX Template

Copy this folder to `src/module-your-name` when starting a business module.

Recommended structure:

- `api/`: module API wrappers.
- `composables/`: hooks for page data and workflows.
- `components/`: module-only UI components.
- `views/`: route page components.
- `router.ts`: module route export used by the base route registry when the module is ready.

Keep shared login, permission, request, upload, table, modal, notification, and layout logic in `src/base`.
