import { lazy, Suspense } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from '@/components/ui/sonner'
import { AuthProvider } from '@/lib/auth'
import { RequireAuth, SetupGate } from '@/lib/guards'
import { Shell } from '@/components/Shell'
import { LoginPage } from '@/pages/LoginPage'
import { SetupPage } from '@/pages/SetupPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { RecipientListsPage } from '@/pages/RecipientListsPage'
import { ApiKeysPage } from '@/pages/ApiKeysPage'
import { IssuesPage } from '@/pages/IssuesPage'
import { IssueDetailPage } from '@/pages/IssueDetailPage'
import { EventsPage } from '@/pages/EventsPage'
import { AlertRulesPage } from '@/pages/AlertRulesPage'
import { AlertsPage } from '@/pages/AlertsPage'
import { ServicesPage } from '@/pages/ServicesPage'
import { NotFoundPage } from '@/pages/NotFoundPage'

// Dashboard pulls in recharts + framer-motion, so load it on demand.
const DashboardPage = lazy(() =>
  import('@/pages/DashboardPage').then((m) => ({ default: m.DashboardPage })),
)

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthProvider>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route
              path="/setup"
              element={
                <RequireAuth>
                  <SetupPage />
                </RequireAuth>
              }
            />
            <Route
              element={
                <RequireAuth>
                  <SetupGate>
                    <Shell />
                  </SetupGate>
                </RequireAuth>
              }
            >
              <Route
                path="/dashboard"
                element={
                  <Suspense fallback={<div className="p-6">Loading dashboard…</div>}>
                    <DashboardPage />
                  </Suspense>
                }
              />
              <Route path="/services" element={<ServicesPage />} />
              <Route path="/settings" element={<SettingsPage />} />
              <Route path="/recipient-lists" element={<RecipientListsPage />} />
              <Route path="/api-keys" element={<ApiKeysPage />} />
              <Route path="/issues" element={<IssuesPage />} />
              <Route path="/issues/:id" element={<IssueDetailPage />} />
              <Route path="/events" element={<EventsPage />} />
              <Route path="/alert-rules" element={<AlertRulesPage />} />
              <Route path="/alerts" element={<AlertsPage />} />
            </Route>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </AuthProvider>
      </BrowserRouter>
      <Toaster />
    </QueryClientProvider>
  )
}
