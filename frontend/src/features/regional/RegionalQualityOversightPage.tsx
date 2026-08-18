import { useQuery } from '@tanstack/react-query'
import { School } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { listSchools } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function RegionalQualityOversightPage() {
  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.schools(),
    queryFn: () => listSchools(),
  })

  const schools = data ?? []

  return (
    <AppShell
      title="Regional Quality Oversight & Compliance"
      description="Monitor school quality scorecards, accreditation status, and compliance flags across your region."
    >
      <div className="space-y-4">
        {isError && (
          <Banner
            variant="error"
            title="Failed to load schools"
            description="Try refreshing the page."
          />
        )}

        <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
          {isLoading ? (
            <div className="flex justify-center py-12">
              <Spinner />
            </div>
          ) : schools.length === 0 ? (
            <EmptyState
              icon={School}
              title="No schools found"
              description="Schools assigned to your region will appear here."
            />
          ) : (
            <table className="w-full text-left text-xs text-slate-600">
              <thead className="border-b border-slate-100 bg-slate-50 text-[11px] font-bold uppercase tracking-wider text-slate-500">
                <tr>
                  <th className="px-4 py-3">School Name</th>
                  <th className="px-4 py-3">School Code</th>
                  <th className="px-4 py-3">Address</th>
                  <th className="px-4 py-3 text-right">Added</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 font-normal">
                {schools.map((s) => (
                  <tr key={s.id} className="hover:bg-slate-50/80">
                    <td className="px-4 py-3 font-bold text-slate-900">{s.name}</td>
                    <td className="px-4 py-3 font-mono text-slate-500">{s.code}</td>
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
