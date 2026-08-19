import { useQuery } from '@tanstack/react-query'
import { HardDrive, Server } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { listSchools } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function RegionalResourcesPage() {
  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.schools(),
    queryFn: () => listSchools(),
  })

  const schools = data ?? []

  return (
    <AppShell
      title="Resource & School Infrastructure"
      description="Overview of schools in your region and their infrastructure status."
    >
      <div className="space-y-4">
        {isError && (
          <Banner
            variant="error"
            title="Failed to load schools"
            description="Try refreshing the page."
          />
        )}

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm flex items-center gap-3">
            <Server className="h-6 w-6 text-teal-600" />
            <div>
              <p className="text-xs font-medium text-slate-500">Schools in Region</p>
              <p className="text-xl font-bold text-slate-900">{schools.length}</p>
            </div>
          </div>
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm flex items-center gap-3">
            <HardDrive className="h-6 w-6 text-slate-600" />
            <div>
              <p className="text-xs font-medium text-slate-500">School Box Sync</p>
              <p className="text-sm font-bold text-slate-500">Managed via admin console</p>
            </div>
          </div>
        </div>

        <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
          {isLoading ? (
            <div className="flex justify-center py-12">
              <Spinner />
            </div>
          ) : schools.length === 0 ? (
            <EmptyState
              icon={Server}
              title="No schools found"
              description="Schools assigned to your region will appear here."
            />
          ) : (
            <table className="w-full text-left text-xs text-slate-600">
              <thead className="border-b border-slate-100 bg-slate-50 text-[11px] font-bold uppercase tracking-wider text-slate-500">
                <tr>
                  <th className="px-4 py-3">School Name</th>
                  <th className="px-4 py-3">Code</th>
                  <th className="px-4 py-3">Address</th>
                  <th className="px-4 py-3 text-right">Registered</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 font-normal">
                {schools.map((s) => (
                  <tr key={s.id} className="hover:bg-slate-50/80">
                    <td className="px-4 py-3 font-semibold text-slate-900">{s.name}</td>
                    <td className="px-4 py-3 font-mono text-teal-700">{s.code}</td>
                    <td className="px-4 py-3 text-slate-500">{s.address ?? '—'}</td>
                    <td className="px-4 py-3 text-right font-mono text-slate-400">
                      {new Date(s.created_at).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </AppShell>
  )
}
