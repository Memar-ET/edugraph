import * as RadixTabs from '@radix-ui/react-tabs'
import type { ReactNode } from 'react'

import { cn } from '@lib/utils/cn'

export const Tabs = RadixTabs.Root

interface TabsListProps {
  children: ReactNode
  className?: string
}

export function TabsList({ children, className }: TabsListProps) {
  return (
    <RadixTabs.List
      className={cn(
        'flex border-b border-gray-200',
        className,
      )}
    >
      {children}
    </RadixTabs.List>
  )
}

interface TabsTriggerProps {
  value: string
  children: ReactNode
  className?: string
}

export function TabsTrigger({ value, children, className }: TabsTriggerProps) {
  return (
    <RadixTabs.Trigger
      value={value}
      className={cn(
        'relative px-4 py-2.5 text-sm font-medium text-gray-500 transition-colors',
        'hover:text-gray-800',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2',
        'data-[state=active]:text-primary-700',
        'data-[state=active]:after:absolute data-[state=active]:after:bottom-0 data-[state=active]:after:left-0 data-[state=active]:after:right-0 data-[state=active]:after:h-0.5 data-[state=active]:after:rounded-t data-[state=active]:after:bg-primary-700',
        className,
      )}
    >
      {children}
    </RadixTabs.Trigger>
  )
}

interface TabsContentProps {
  value: string
  children: ReactNode
  className?: string
}

export function TabsContent({ value, children, className }: TabsContentProps) {
  return (
    <RadixTabs.Content
      value={value}
      className={cn('focus:outline-none', className)}
    >
      {children}
    </RadixTabs.Content>
  )
}
