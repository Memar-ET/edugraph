import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Briefcase, Plus } from 'lucide-react'
import { useState } from 'react'
import type { FormEvent } from 'react'

import { AppShell } from '@components/layout'
import {
  Banner,
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  EmptyState,
  Input,
  Label,
  Spinner,
  StatusPill,
} from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { createCareerPath, listCareerPaths } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'

export function CareerPathsPage() {
  const user = useAuthStore((s) => s.user)
  const canCreate = user?.role === 'ministry_admin'
  const queryClient = useQueryClient()

  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.careerPaths(),
    queryFn: listCareerPaths,
  })

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [subjects, setSubjects] = useState('')

  const create = useMutation({
    mutationFn: createCareerPath,
    onSuccess: () => {
      setTitle('')
      setDescription('')
      setSubjects('')
      void queryClient.invalidateQueries({ queryKey: queryKeys.careerPaths() })
    },
  })

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (!title.trim()) return
    create.mutate({
      title: title.trim(),
      description: description.trim() || undefined,
      required_subjects: subjects
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
    })
  }

  return (
    <AppShell title="Career paths" description="Paths matched against a student's subject strengths.">
      <div className="grid gap-6 lg:grid-cols-3">
        <div className="space-y-4 lg:col-span-2">
          {isLoading && (
            <div className="flex items-center gap-2 text-sm text-gray-500">
              <Spinner /> Loading career paths…
            </div>
          )}
          {isError && <Banner tone="error">{apiErrorMessage(error, 'Could not load career paths.')}</Banner>}
          {data && data.length === 0 && (
            <EmptyState icon={Briefcase} title="No career paths defined yet" />
          )}
          {data && data.length > 0 && (
            <div className="grid gap-4 sm:grid-cols-2">
              {data.map((path) => (
                <Card key={path.id}>
                  <CardHeader>
                    <CardTitle className="text-base">{path.title}</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    {path.description && <p className="text-sm text-gray-600">{path.description}</p>}
                    {path.required_subjects && path.required_subjects.length > 0 && (
                      <div className="flex flex-wrap gap-1">
                        {path.required_subjects.map((s) => (
                          <StatusPill key={s} tone="primary">
                            {s}
                          </StatusPill>
                        ))}
                      </div>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>

        {canCreate && (
          <Card className="h-fit">
            <CardHeader>
              <CardTitle className="text-base">Add a career path</CardTitle>
            </CardHeader>
            <CardContent>
              <form className="space-y-3" onSubmit={handleSubmit}>
                {create.isError && (
                  <Banner tone="error">{apiErrorMessage(create.error, 'Could not create career path.')}</Banner>
                )}
                <div>
                  <Label htmlFor="career-title">Title</Label>
                  <Input id="career-title" value={title} onChange={(e) => setTitle(e.target.value)} required />
                </div>
                <div>
                  <Label htmlFor="career-desc">Description</Label>
                  <Input id="career-desc" value={description} onChange={(e) => setDescription(e.target.value)} />
                </div>
                <div>
                  <Label htmlFor="career-subjects">Required subjects (comma-separated)</Label>
                  <Input
                    id="career-subjects"
                    value={subjects}
                    onChange={(e) => setSubjects(e.target.value)}
                    placeholder="PHY, MATH"
                  />
                </div>
                <Button type="submit" isLoading={create.isPending} className="w-full">
                  <Plus className="h-4 w-4" aria-hidden />
                  Add path
                </Button>
              </form>
            </CardContent>
          </Card>
        )}
      </div>
    </AppShell>
  )
}
