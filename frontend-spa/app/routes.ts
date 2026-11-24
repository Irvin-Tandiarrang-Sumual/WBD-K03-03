import { type RouteConfig, index, layout, prefix, route } from "@react-router/dev/routes";

export default [
  // Auth routes with minimalist backdrop layout
  layout("layouts/auth-layout.tsx", [
    route("login", "routes/login.tsx"),
  ]),

  // Main console workspace layout
  layout("layouts/app-layout.tsx", [
    index("routes/home.tsx"),
    route("databases", "routes/databases.tsx"),
    route("databases/:id", "routes/database-detail.tsx"),
    route("databases/:id/drafts/new", "routes/create-draft.tsx"),
    route("drafts", "routes/draft-list.tsx"),
    route("drafts/:id", "routes/draft-detail.tsx"),
    route("approvals", "routes/approvals.tsx"),

    // Administration section
    ...prefix("admin", [
      route("databases", "routes/admin-databases.tsx"),
      route("users", "routes/admin-users.tsx"),
      route("groups", "routes/admin-groups.tsx"),
      route("groups/:id", "routes/admin-group-detail.tsx"),
    ]),
  ]),
] satisfies RouteConfig;
