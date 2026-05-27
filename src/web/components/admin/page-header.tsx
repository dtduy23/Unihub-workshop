'use client'

import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface PageHeaderProps {
  title: string
  description?: string
  actionLabel?: string
  onAction?: () => void
}

export function PageHeader({ 
  title, 
  description, 
  actionLabel, 
  onAction 
}: PageHeaderProps) {
  return (
    <div className="flex items-center justify-between">
      <div>
        <h1 className="text-3xl font-bold tracking-tight text-slate-900">
          {title}
        </h1>
        {description && (
          <p className="mt-1 text-muted-foreground">
            {description}
          </p>
        )}
      </div>
      {actionLabel && (
        <Button 
          onClick={onAction}
          className="bg-primary hover:bg-primary/90 shadow-lg shadow-primary/20 h-11 px-6 rounded-xl transition-all duration-200 active:scale-95"
        >
          <Plus className="mr-2 h-4 w-4" />
          {actionLabel}
        </Button>
      )}
    </div>
  )
}
