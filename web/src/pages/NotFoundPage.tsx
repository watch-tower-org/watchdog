import { Link } from 'react-router-dom'
import { Compass } from 'lucide-react'
import { Button } from '@/components/ui/button'

export function NotFoundPage() {
  return (
    <div className="flex h-screen flex-col items-center justify-center gap-4 bg-background">
      <Compass className="h-12 w-12 text-muted-foreground" />
      <div className="text-center">
        <h1 className="font-mono text-4xl font-bold">404</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          This page is beyond the perimeter.
        </p>
      </div>
      <Button asChild>
        <Link to="/dashboard">Back to dashboard</Link>
      </Button>
    </div>
  )
}
