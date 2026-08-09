import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Check, Loader2, Radar, Send } from 'lucide-react'
import { getApi, getErrorMessage } from '@/lib/api'
import type { EmailSettings, Settings } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

interface SetupForm {
  smtp_host: string
  smtp_port: string
  smtp_username: string
  smtp_password: string
  smtp_from_email: string
  smtp_from_name: string
}

const emptyForm: SetupForm = {
  smtp_host: '',
  smtp_port: '587',
  smtp_username: '',
  smtp_password: '',
  smtp_from_email: '',
  smtp_from_name: 'WatchTower',
}

export function SetupPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [form, setForm] = useState<SetupForm>(emptyForm)
  const [testEmail, setTestEmail] = useState('')

  const { data } = useQuery({
    queryKey: ['settings'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: Settings }>('/settings')
      return data.data
    },
  })

  useEffect(() => {
    if (data?.setup_complete) {
      navigate('/dashboard', { replace: true })
    }
  }, [data, navigate])

  const set = (key: keyof SetupForm, value: string) =>
    setForm((f) => ({ ...f, [key]: value }))

  const saveMutation = useMutation({
    mutationFn: async () => {
      await getApi().put('/settings/email', {
        smtp_host: form.smtp_host,
        smtp_port: form.smtp_port ? Number(form.smtp_port) : undefined,
        smtp_username: form.smtp_username,
        smtp_password: form.smtp_password || undefined,
        smtp_from_email: form.smtp_from_email,
        smtp_from_name: form.smtp_from_name,
      })
      await getApi().put('/settings', { setup_complete: true })
    },
    onSuccess: () => {
      toast.success('Setup complete!')
      queryClient.invalidateQueries({ queryKey: ['settings'] })
      navigate('/dashboard', { replace: true })
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  const testMutation = useMutation({
    mutationFn: async () => {
      const { data } = await getApi().post<{ data: EmailSettings }>(
        '/settings/email/test',
        { to: testEmail },
      )
      return data
    },
    onSuccess: () => toast.success('Test email sent successfully'),
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  const canSave = form.smtp_host.trim() && form.smtp_from_email.trim()

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <div className="w-full max-w-lg space-y-6">
        <div className="flex flex-col items-center gap-3 text-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            <Radar className="h-8 w-8" />
          </div>
          <div>
            <h1 className="font-mono text-2xl font-bold">Welcome to WatchTower</h1>
            <p className="text-sm text-muted-foreground">
              Let&apos;s get your email alerting configured.
            </p>
          </div>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Check className="h-5 w-5 text-primary" />
              Step 1 — SMTP configuration
            </CardTitle>
            <CardDescription>
              Used to send alert emails. You can test your settings before finishing.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="setup-host">SMTP host</Label>
                <Input
                  id="setup-host"
                  value={form.smtp_host}
                  onChange={(e) => set('smtp_host', e.target.value)}
                  placeholder="smtp.example.com"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="setup-port">Port</Label>
                <Input
                  id="setup-port"
                  type="number"
                  value={form.smtp_port}
                  onChange={(e) => set('smtp_port', e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="setup-user">Username</Label>
                <Input
                  id="setup-user"
                  value={form.smtp_username}
                  onChange={(e) => set('smtp_username', e.target.value)}
                  autoComplete="off"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="setup-pass">Password</Label>
                <Input
                  id="setup-pass"
                  type="password"
                  value={form.smtp_password}
                  onChange={(e) => set('smtp_password', e.target.value)}
                  autoComplete="new-password"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="setup-from">From email</Label>
                <Input
                  id="setup-from"
                  type="email"
                  value={form.smtp_from_email}
                  onChange={(e) => set('smtp_from_email', e.target.value)}
                  placeholder="watchtower@example.com"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="setup-from-name">From name</Label>
                <Input
                  id="setup-from-name"
                  value={form.smtp_from_name}
                  onChange={(e) => set('smtp_from_name', e.target.value)}
                />
              </div>
            </div>

            <div className="flex items-end gap-2 border-t border-border pt-4">
              <div className="flex-1 space-y-2">
                <Label htmlFor="setup-test-email">Test to</Label>
                <Input
                  id="setup-test-email"
                  type="email"
                  value={testEmail}
                  onChange={(e) => setTestEmail(e.target.value)}
                  placeholder="you@example.com"
                />
              </div>
              <Button
                variant="outline"
                onClick={() => testEmail && testMutation.mutate()}
                disabled={!testEmail || testMutation.isPending}
              >
                {testMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Send className="h-4 w-4" />
                )}
                Test
              </Button>
            </div>
          </CardContent>
        </Card>

        <Button
          className="w-full"
          size="lg"
          onClick={() => saveMutation.mutate()}
          disabled={!canSave || saveMutation.isPending}
        >
          {saveMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
          Finish setup
        </Button>
      </div>
    </div>
  )
}
