import { createRootRoute, createRoute, createRouter, Outlet } from '@tanstack/react-router'
import { RootLayout } from '@/components/layout/RootLayout'
import { DashboardPage } from '@/routes/DashboardPage'
import { NotFoundPage } from '@/routes/NotFoundPage'

const rootRoute = createRootRoute({
  component: RootLayout,
  notFoundComponent: NotFoundPage,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: DashboardPage,
})

export const routeTree = rootRoute.addChildren([indexRoute])

declare module '@tanstack/react-router' {
  interface Register {
    router: ReturnType<typeof createRouter<typeof routeTree>>
  }
}

export { Outlet }
