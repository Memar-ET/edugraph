import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'

import { AppShell } from '@components/layout'
import { Card, CardContent, CardHeader, CardTitle, useToast } from '@components/ui'
import { changePassword } from '@lib/api/endpoints'
import { useAuthStore } from '@stores/auth.store'

export function SettingsPage() {
  const { toast } = useToast()
  const user = useAuthStore((s) => s.user)
  const [oldPw, setOldPw] = useState('')
  const [newPw, setNewPw] = useState('')
  const [confirm, setConfirm] = useState('')

  const changePwMutation = useMutation({
    mutationFn: () => changePassword({ currentPassword: oldPw, newPassword: newPw }),
    onSuccess: () => {
      toast({ title: 'Password changed', variant: 'success' })
      setOldPw('')
      setNewPw('')
      setConfirm('')
    },
    onError: () => {
      toast({ title: 'Failed to change password', description: 'Check your current password and try again.', variant: 'error' })
    },
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (newPw !== confirm) {
      toast({ title: 'Passwords do not match', variant: 'error' })
      return
    }
    changePwMutation.mutate()
  }

  return (
    <AppShell>
      <div className="space-y-6 max-w-lg">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Settings</h1>
          <p className="mt-1 text-sm text-gray-500">Manage your account preferences.</p>
        </div>

        {/* Profile card */}
        <Card>
          <CardHeader><CardTitle>Profile</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm text-gray-700">
            <div className="flex gap-2">
              <span className="w-24 shrink-0 font-medium text-gray-500">Email</span>
              <span>{user?.email ?? '—'}</span>
            </div>
            <div className="flex gap-2">
              <span className="w-24 shrink-0 font-medium text-gray-500">Role</span>
              <span className="capitalize">{user?.role?.replace('_', ' ') ?? '—'}</span>
            </div>
          </CardContent>
        </Card>

        {/* Change password */}
        <Card>
          <CardHeader><CardTitle>Change Password</CardTitle></CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-1">
                <label htmlFor="old-pw" className="block text-sm font-medium text-gray-700">
                  Current password
                </label>
                <input
                  id="old-pw"
                  type="password"
                  value={oldPw}
                  onChange={(e) => setOldPw(e.target.value)}
                  required
                  autoComplete="current-password"
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>
              <div className="space-y-1">
                <label htmlFor="new-pw" className="block text-sm font-medium text-gray-700">
                  New password
                </label>
                <input
                  id="new-pw"
                  type="password"
                  value={newPw}
                  onChange={(e) => setNewPw(e.target.value)}
                  required
                  minLength={8}
                  autoComplete="new-password"
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>
              <div className="space-y-1">
                <label htmlFor="confirm-pw" className="block text-sm font-medium text-gray-700">
                  Confirm new password
                </label>
                <input
                  id="confirm-pw"
                  type="password"
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  required
                  autoComplete="new-password"
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>
              <button
                type="submit"
                disabled={changePwMutation.isPending}
                className="rounded-lg bg-primary-700 px-4 py-2 text-sm font-medium text-white hover:bg-primary-600 focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:opacity-50"
              >
                {changePwMutation.isPending ? 'Saving…' : 'Save password'}
              </button>
            </form>
          </CardContent>
        </Card>
      </div>
    </AppShell>
  )
}
