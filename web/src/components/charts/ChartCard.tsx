import type { ReactNode } from 'react'
import { motion } from 'framer-motion'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { cn } from '@/lib/utils'

interface ChartCardProps {
  title: string
  description?: string
  icon?: ReactNode
  action?: ReactNode
  /** Entrance animation delay in seconds (for staggering card rows). */
  delay?: number
  className?: string
  children: ReactNode
}

/** Animated card shell used across the dashboard for charts and feeds. */
export function ChartCard({
  title,
  description,
  icon,
  action,
  delay = 0,
  className,
  children,
}: ChartCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.45, delay, ease: 'easeOut' }}
      className={cn('h-full', className)}
    >
      <Card className="h-full transition-colors hover:border-primary/40">
        <CardHeader>
          <div className="flex items-start justify-between gap-3">
            <div className="flex min-w-0 items-center gap-2">
              {icon && (
                <span className="shrink-0 rounded-md border bg-muted/60 p-1.5 text-primary">
                  {icon}
                </span>
              )}
              <CardTitle className="text-sm">{title}</CardTitle>
            </div>
            {action}
          </div>
          {description && (
            <CardDescription>{description}</CardDescription>
          )}
        </CardHeader>
        <CardContent>{children}</CardContent>
      </Card>
    </motion.div>
  )
}
