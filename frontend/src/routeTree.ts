import { createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { RootLayout } from '@/components/layout/RootLayout'
import { AppShell } from '@/components/layout/AppShell'
import { DocsLayout } from '@/routes/docs/DocsLayout'
import { DashboardPage } from '@/routes/DashboardPage'
import { NotFoundPage } from '@/routes/NotFoundPage'
// Developer
import { AppRequestsPage } from '@/routes/developer/AppRequestsPage'
import { AppRequestDetailPage } from '@/routes/developer/AppRequestDetailPage'
import { NewAppRequestPage } from '@/routes/developer/NewAppRequestPage'
// Admin
import { AdminAppsPage } from '@/routes/admin/AdminAppsPage'
import { AdminUsersPage } from '@/routes/admin/AdminUsersPage'
import { AdminAuditLogsPage } from '@/routes/admin/AdminAuditLogsPage'
import { AdminAppRequestsPage } from '@/routes/admin/AdminAppRequestsPage'
// Account
import { AccountOverviewPage } from '@/routes/account/AccountOverviewPage'
// Docs
import { DocsOverviewPage } from '@/routes/docs/DocsOverviewPage'
import { DocsStartPage } from '@/routes/docs/DocsStartPage'
import { DocsConceptsPage } from '@/routes/docs/DocsConceptsPage'
import { DocsSdkPage } from '@/routes/docs/DocsSdkPage'
import { DocsManagementPage } from '@/routes/docs/DocsManagementPage'
import { DocsAccountPage } from '@/routes/docs/DocsAccountPage'
import { DocsSecurityPage } from '@/routes/docs/DocsSecurityPage'
import { DocsApiPage } from '@/routes/docs/DocsApiPage'

const rootRoute = createRootRoute({
  component: RootLayout,
  notFoundComponent: NotFoundPage,
})

// Shell layout (sidebar)
const shellRoute = createRoute({ getParentRoute: () => rootRoute, id: 'shell', component: AppShell })

// Public index
const indexRoute = createRoute({ getParentRoute: () => rootRoute, path: '/', component: DashboardPage })

// Developer routes
const devRoute = createRoute({ getParentRoute: () => shellRoute, path: '/developer' })
const devAppRequestsRoute = createRoute({ getParentRoute: () => devRoute, path: '/app-requests', component: AppRequestsPage })
const devAppRequestsNewRoute = createRoute({ getParentRoute: () => devRoute, path: '/app-requests/new', component: NewAppRequestPage })
const devAppRequestDetailRoute = createRoute({ getParentRoute: () => devRoute, path: '/app-requests/$id', component: AppRequestDetailPage })

// Admin routes
const adminRoute = createRoute({ getParentRoute: () => shellRoute, path: '/admin' })
const adminAppsRoute = createRoute({ getParentRoute: () => adminRoute, path: '/apps', component: AdminAppsPage })
const adminUsersRoute = createRoute({ getParentRoute: () => adminRoute, path: '/users', component: AdminUsersPage })
const adminAuditRoute = createRoute({ getParentRoute: () => adminRoute, path: '/audit-logs', component: AdminAuditLogsPage })
const adminAppRequestsRoute = createRoute({ getParentRoute: () => adminRoute, path: '/app-requests', component: AdminAppRequestsPage })

// Account routes
const accountRoute = createRoute({ getParentRoute: () => shellRoute, path: '/account' })
const accountOverviewRoute = createRoute({ getParentRoute: () => accountRoute, path: '/', component: AccountOverviewPage })

// Docs routes (separate layout, no app shell)
const docsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/docs', component: DocsLayout })
const docsIndexRoute = createRoute({ getParentRoute: () => docsRoute, path: '/', component: DocsOverviewPage })
const docsStartRoute = createRoute({ getParentRoute: () => docsRoute, path: '/start', component: DocsStartPage })
const docsConceptsRoute = createRoute({ getParentRoute: () => docsRoute, path: '/concepts', component: DocsConceptsPage })
const docsSdkRoute = createRoute({ getParentRoute: () => docsRoute, path: '/sdk', component: DocsSdkPage })
const docsManagementRoute = createRoute({ getParentRoute: () => docsRoute, path: '/management', component: DocsManagementPage })
const docsAccountRoute = createRoute({ getParentRoute: () => docsRoute, path: '/account', component: DocsAccountPage })
const docsSecurityRoute = createRoute({ getParentRoute: () => docsRoute, path: '/security', component: DocsSecurityPage })
const docsApiRoute = createRoute({ getParentRoute: () => docsRoute, path: '/api', component: DocsApiPage })

export const routeTree = rootRoute.addChildren([
  indexRoute,
  shellRoute.addChildren([
    devRoute.addChildren([devAppRequestsRoute, devAppRequestsNewRoute, devAppRequestDetailRoute]),
    adminRoute.addChildren([adminAppsRoute, adminUsersRoute, adminAuditRoute, adminAppRequestsRoute]),
    accountRoute.addChildren([accountOverviewRoute]),
  ]),
  docsRoute.addChildren([
    docsIndexRoute,
    docsStartRoute,
    docsConceptsRoute,
    docsSdkRoute,
    docsManagementRoute,
    docsAccountRoute,
    docsSecurityRoute,
    docsApiRoute,
  ]),
])

declare module '@tanstack/react-router' {
  interface Register {
    router: ReturnType<typeof createRouter<typeof routeTree>>
  }
}
