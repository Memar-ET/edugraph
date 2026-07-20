import { useMutation } from '@tanstack/react-query'
import { MessageCircleQuestion, Send } from 'lucide-react'
import { useState } from 'react'
import type { ComponentPropsWithoutRef, KeyboardEvent } from 'react'
import ReactMarkdown from 'react-markdown'

import { AppShell } from '@components/layout'
import { Banner, Button, Card, CardContent, EmptyState, Select, Spinner, StatusPill } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { askTutor } from '@lib/api/endpoints'
import { extractLabels } from '@lib/utils/format'
import type { TutorAskResponse } from '@/types/api'

// Gemini answers come back as Markdown (headings, bold, lists) -- render it
// as such instead of dumping raw ###/** into the chat bubble. Overrides
// keep the type scale in step with the rest of the app rather than
// react-markdown's default browser styles.
const MARKDOWN_COMPONENTS = {
  h1: (p: ComponentPropsWithoutRef<'h1'>) => <h3 className="mt-3 text-base font-semibold text-gray-900 first:mt-0" {...p} />,
  h2: (p: ComponentPropsWithoutRef<'h2'>) => <h3 className="mt-3 text-base font-semibold text-gray-900 first:mt-0" {...p} />,
  h3: (p: ComponentPropsWithoutRef<'h3'>) => <h4 className="mt-3 text-sm font-semibold text-gray-900 first:mt-0" {...p} />,
  h4: (p: ComponentPropsWithoutRef<'h4'>) => <h4 className="mt-3 text-sm font-semibold text-gray-900 first:mt-0" {...p} />,
  p: (p: ComponentPropsWithoutRef<'p'>) => <p className="text-sm text-gray-800" {...p} />,
  strong: (p: ComponentPropsWithoutRef<'strong'>) => <strong className="font-semibold text-gray-900" {...p} />,
  ul: (p: ComponentPropsWithoutRef<'ul'>) => <ul className="list-inside list-disc space-y-1 text-sm text-gray-800" {...p} />,
  ol: (p: ComponentPropsWithoutRef<'ol'>) => <ol className="list-inside list-decimal space-y-1 text-sm text-gray-800" {...p} />,
  hr: () => <hr className="my-3 border-gray-200" />,
  code: (p: ComponentPropsWithoutRef<'code'>) => <code className="rounded bg-gray-100 px-1 py-0.5 font-mono text-xs" {...p} />,
}

interface ChatTurn {
  question: string
  response?: TutorAskResponse
}

export function TutorPage() {
  const [turns, setTurns] = useState<ChatTurn[]>([])
  const [question, setQuestion] = useState('')
  const [language, setLanguage] = useState<'en' | 'am'>('en')

  const ask = useMutation({
    mutationFn: (q: string) => askTutor({ question: q, language }),
    onSuccess: (response, q) => {
      setTurns((prev) => [...prev, { question: q, response }])
      setQuestion('')
    },
  })

  const handleSend = () => {
    const trimmed = question.trim()
    if (trimmed.length < 3 || ask.isPending) return
    ask.mutate(trimmed)
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <AppShell
      title="Ask the tutor"
      description="Personalized explanations based on your recent exam gaps."
      actions={
        <Select
          value={language}
          onChange={(e) => setLanguage(e.target.value as 'en' | 'am')}
          className="w-32"
          aria-label="Answer language"
        >
          <option value="en">English</option>
          <option value="am">Amharic</option>
        </Select>
      }
    >
      <div className="mx-auto flex max-w-2xl flex-col gap-4">
        {turns.length === 0 && !ask.isPending && (
          <EmptyState
            icon={MessageCircleQuestion}
            title="Ask anything about a topic you're stuck on"
            description="The tutor uses your gap records and the curriculum graph to tailor its answer to what you actually missed."
          />
        )}

        <div className="space-y-4">
          {turns.map((turn, i) => (
            <div key={i} className="space-y-2">
              <div className="ml-auto max-w-[85%] rounded-lg rounded-tr-sm bg-primary-700 px-4 py-2 text-sm text-white">
                {turn.question}
              </div>
              {turn.response && (
                <Card className="max-w-[90%]">
                  <CardContent className="space-y-3 pt-4">
                    <div className="space-y-2">
                      <ReactMarkdown components={MARKDOWN_COMPONENTS}>{turn.response.answer}</ReactMarkdown>
                    </div>
                    {extractLabels(turn.response.relatedTopics).length > 0 && (
                      <div>
                        <p className="mb-1 text-xs font-medium uppercase tracking-wide text-gray-400">
                          Related topics
                        </p>
                        <div className="flex flex-wrap gap-1">
                          {extractLabels(turn.response.relatedTopics).map((t) => (
                            <StatusPill key={t} tone="primary">
                              {t}
                            </StatusPill>
                          ))}
                        </div>
                      </div>
                    )}
                    <p className="text-right text-[11px] text-gray-300">{turn.response.model}</p>
                  </CardContent>
                </Card>
              )}
            </div>
          ))}
          {ask.isPending && (
            <div className="flex items-center gap-2 text-sm text-gray-500">
              <Spinner /> Thinking…
            </div>
          )}
          {ask.isError && (
            <Banner tone="error">{apiErrorMessage(ask.error, 'The tutor could not answer that. Try again.')}</Banner>
          )}
        </div>

        <form
          className="sticky bottom-4 flex items-end gap-2 rounded-lg border border-gray-200 bg-white p-2 shadow-stamp"
          onSubmit={(e) => {
            e.preventDefault()
            handleSend()
          }}
        >
          <textarea
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Ask a question about something you're stuck on…"
            rows={2}
            className="flex-1 resize-none rounded-md border-0 px-2 py-1.5 text-sm focus:outline-none focus:ring-0"
          />
          <Button type="submit" isLoading={ask.isPending} disabled={question.trim().length < 3}>
            <Send className="h-4 w-4" aria-hidden />
            Ask
          </Button>
        </form>
      </div>
    </AppShell>
  )
}
