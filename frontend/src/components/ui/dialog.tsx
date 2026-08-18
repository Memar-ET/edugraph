import * as RadixDialog from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'

import { cn } from '@lib/utils/cn'

export const Dialog = RadixDialog.Root
export const DialogTrigger = RadixDialog.Trigger
export const DialogClose = RadixDialog.Close

interface DialogContentProps {
  title: string
  description?: string
  children: ReactNode
  className?: string
}

export function DialogContent({ title, description, children, className }: DialogContentProps) {
  return (
    <RadixDialog.Portal>
      <RadixDialog.Overlay className="fixed inset-0 z-40 bg-black/40 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />
      <RadixDialog.Content
        className={cn(
          'fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2',
          'rounded-xl border border-gray-200 bg-white shadow-xl',
          'focus:outline-none',
          'data-[state=open]:animate-in data-[state=closed]:animate-out',
          'data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
          'data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95',
          className,
        )}
      >
        <div className="flex items-center justify-between border-b border-gray-100 px-6 py-4">
          <RadixDialog.Title className="text-base font-semibold text-gray-900">
            {title}
          </RadixDialog.Title>
          <RadixDialog.Close className="rounded p-1 text-gray-400 hover:text-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-500">
            <X className="h-4 w-4" aria-hidden />
            <span className="sr-only">Close</span>
          </RadixDialog.Close>
        </div>
        {description && (
          <RadixDialog.Description className="px-6 pt-4 text-sm text-gray-500">
            {description}
          </RadixDialog.Description>
        )}
        <div className="px-6 py-4">{children}</div>
      </RadixDialog.Content>
    </RadixDialog.Portal>
  )
}

interface ConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description: string
  confirmLabel?: string
  cancelLabel?: string
  onConfirm: () => void
  destructive?: boolean
  loading?: boolean
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  onConfirm,
  destructive = false,
  loading = false,
}: ConfirmDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title={title} description={description}>
        <div className="flex justify-end gap-2 pt-2">
          <DialogClose asChild>
            <button className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500">
              {cancelLabel}
            </button>
          </DialogClose>
          <button
            onClick={onConfirm}
            disabled={loading}
            className={cn(
              'rounded-lg px-4 py-2 text-sm font-medium text-white focus:outline-none focus:ring-2 focus:ring-offset-1 disabled:opacity-50',
              destructive
                ? 'bg-alert-600 hover:bg-alert-700 focus:ring-alert-500'
                : 'bg-primary-700 hover:bg-primary-600 focus:ring-primary-500',
            )}
          >
            {loading ? 'Loading…' : confirmLabel}
          </button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
