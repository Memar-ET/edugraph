import * as RadixToast from '@radix-ui/react-toast'
import { X } from 'lucide-react'
import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'

import { cn } from '@lib/utils/cn'

type ToastVariant = 'default' | 'success' | 'error' | 'warning'

interface ToastItem {
  id: string
  title: string
  description?: string
  variant?: ToastVariant
}

interface ToastContextValue {
  toast: (opts: Omit<ToastItem, 'id'>) => void
}

const ToastContext = createContext<ToastContextValue>({ toast: () => {} })

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([])

  const toast = useCallback((opts: Omit<ToastItem, 'id'>) => {
    const id = Math.random().toString(36).slice(2)
    setToasts((prev) => [...prev, { id, ...opts }])
  }, [])

  const remove = (id: string) => setToasts((prev) => prev.filter((t) => t.id !== id))

  return (
    <ToastContext.Provider value={{ toast }}>
      <RadixToast.Provider swipeDirection="right">
        {children}
        {toasts.map((t) => (
          <RadixToast.Root
            key={t.id}
            open
            onOpenChange={(open) => !open && remove(t.id)}
            duration={4000}
            className={cn(
              'flex items-start gap-3 rounded-lg border px-4 py-3 shadow-md',
              'data-[state=open]:animate-in data-[state=closed]:animate-out',
              'data-[swipe=end]:animate-out data-[swipe=end]:translate-x-[var(--radix-toast-swipe-end-x)]',
              t.variant === 'success' && 'border-health-200 bg-health-50 text-health-800',
              t.variant === 'error' && 'border-alert-200 bg-alert-50 text-alert-800',
              t.variant === 'warning' && 'border-seal-200 bg-seal-50 text-seal-800',
              (!t.variant || t.variant === 'default') && 'border-gray-200 bg-white text-gray-900',
            )}
          >
            <div className="flex-1">
              <RadixToast.Title className="text-sm font-medium">{t.title}</RadixToast.Title>
              {t.description && (
                <RadixToast.Description className="mt-0.5 text-xs opacity-80">
                  {t.description}
                </RadixToast.Description>
              )}
            </div>
            <RadixToast.Close
              onClick={() => remove(t.id)}
              className="rounded p-0.5 opacity-60 hover:opacity-100 focus:outline-none focus:ring-1 focus:ring-primary-500"
            >
              <X className="h-3.5 w-3.5" aria-hidden />
              <span className="sr-only">Dismiss</span>
            </RadixToast.Close>
          </RadixToast.Root>
        ))}
        <RadixToast.Viewport className="fixed bottom-4 right-4 z-[100] flex w-80 flex-col gap-2 outline-none" />
      </RadixToast.Provider>
    </ToastContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components -- hook must live alongside its context
export function useToast() {
  return useContext(ToastContext)
}
