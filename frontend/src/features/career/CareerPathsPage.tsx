import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Briefcase, Plus, Sparkles } from 'lucide-react'
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
  toneForPct,
} from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { createCareerPath, generateCareerMatches, getCareerMatches, listCareerPaths } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'

function MyCareerMatches() {
  const queryClient = useQueryClient()

  // No GET /students/me exists, and none is needed here: the backend
  // resolves the caller's own students.id server-side from the JWT
  // (career/handler.go), so this never needs the roster-browse lookup
  // useMyStudentRecord used to do via GET /students?school_id=... --
  // an endpoint the student role was deliberately never granted (see
  // router.go's RBAC comment on that route), which made this component
  // 403 for every student before matchesQuery ever ran. The query key
  // just needs to be stable per logged-in student, which 'me' already is.
  const matchesQuery = useQuery({
    queryKey: queryKeys.careerMatches('me'),
    queryFn: () => getCareerMatches(),
  })

  const generate = useMutation({
    mutationFn: () => generateCareerMatches(),
    onSuccess: (matches) => {
      queryClient.setQueryData(queryKeys.careerMatches('me'), matches)
    },
  })

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-base">My career matches</CardTitle>
        <Button size="sm" onClick={() => generate.mutate()} isLoading={generate.isPending}>
          <Sparkles className="h-4 w-4" aria-hidden />
          {matchesQuery.data && matchesQuery.data.length > 0 ? 'Regenerate' : 'Generate my matches'}
        </Button>
      </CardHeader>
      <CardContent className="space-y-3">
        {generate.isError && (
          <Banner tone="error">{apiErrorMessage(generate.error, 'Could not generate career matches.')}</Banner>
        )}
        {matchesQuery.isError && (
          <Banner tone="error">{apiErrorMessage(matchesQuery.error, 'Could not load your career matches.')}</Banner>
        )}
        {matchesQuery.isLoading && (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <Spinner /> Loading your matches…
          </div>
        )}
        {matchesQuery.data && matchesQuery.data.length === 0 && (
          <p className="text-sm text-gray-500">
            No matches yet — based on your exam scores across subjects, generate matches to see which career paths
            fit best so far.
          </p>
        )}
        {matchesQuery.data && matchesQuery.data.length > 0 && (
          <ul className="space-y-2">
            {matchesQuery.data.map((m) => (
              <li key={m.career_path_id} className="flex items-center justify-between rounded-md border p-2">
                <span className="text-sm font-medium">{m.title}</span>
                <StatusPill tone={toneForPct(m.score * 100)}>{Math.round(m.score * 100)}%</StatusPill>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}

export function CareerPathsPage() {
  const user = useAuthStore((s) => s.user)
  const canCreate = user?.role === 'ministry_admin'
  const isStudent = user?.role === 'student'
  const queryClient = useQueryClient()

  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.careerPaths(),
    queryFn: listCareerPaths,
  })

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [sector, setSector] = useState('')
  const [minEduLevel, setMinEduLevel] = useState('')

  const create = useMutation({
    mutationFn: createCareerPath,
    onSuccess: () => {
      setTitle('')
      setDescription('')
      setSector('')
      setMinEduLevel('')
      void queryClient.invalidateQueries({ queryKey: queryKeys.careerPaths() })
    },
  })

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (!title.trim() || !sector.trim() || !minEduLevel.trim()) return
    create.mutate({
      title: title.trim(),
      description: description.trim() || undefined,
      sector: sector.trim(),
      minEduLevel: minEduLevel.trim(),
    })
  }

  return (
    <AppShell title="Career paths" description="Paths matched against a student's subject strengths.">
      <div className="grid gap-6 lg:grid-cols-3">
        <div className="space-y-4 lg:col-span-2">
          {isStudent && <MyCareerMatches />}
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
                  <Label htmlFor="career-sector">Sector</Label>
                  <Input
                    id="career-sector"
                    value={sector}
                    onChange={(e) => setSector(e.target.value)}
                    placeholder="Healthcare"
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="career-min-edu">Minimum education level</Label>
                  <Input
                    id="career-min-edu"
                    value={minEduLevel}
                    onChange={(e) => setMinEduLevel(e.target.value)}
                    placeholder="Bachelor's degree"
                    required
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
