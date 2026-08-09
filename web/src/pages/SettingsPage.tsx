import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Loader2, Mail, BellRing, Send } from 'lucide-react'
import { getApi, getErrorMessage } from '@/lib/api'
import type { AlertSettings, EmailSettings } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

export function SettingsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Settings</h1>
        <p className="text-sm text-muted-foreground">
          Configure email delivery and alert behavior.
        </p>
      </div>

      <Tabs defaultValue="email" className="w-full">
        <TabsList>
          <TabsTrigger value="email">
            <Mail className="mr-2 h-4 w-4" />
            Email / SMTP
          </TabsTrigger>
          <TabsTrigger value="alert">
            <BellRing className="mr-2 h-4 w-4" />
            Alert throttling
          </TabsTrigger>
        </TabsList>
        <TabsContent value="email">
          <EmailSettingsTab />
        </TabsContent>
        <TabsContent value="alert">
          <AlertSettingsTab />
        </TabsContent>
      </Tabs>
    </div>
  )
}

function EmailSettingsTab() {
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['email-settings'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: EmailSettings }>('/settings/email')
      return data.data
    },
  })

  const [form, setForm] = useState<{
    smtp_host: string
    smtp_port: string
    smtp_username: string
    smtp_password: string
    smtp_from_email: string
    smtp_from_name: string
  } | null>(null)

  const [testEmail, setTestEmail] = useState('')

  const syncForm = (s?: EmailSettings) =>
    setForm({
      smtp_host: s?.smtp_host ?? '',
      smtp_port: s?.smtp_port ? String(s.smtp_port) : '587',
      smtp_username: s?.smtp_username ?? '',
      smtp_password: '',
      smtp_from_email: s?.smtp_from_email ?? '',
      smtp_from_name: s?.smtp_from_name ?? '',
    })

  useEffect(() => {
    if (data && !form) {
      syncForm(data)
    }
  }, [data, form])

  const updateMutation = useMutation({
    mutationFn: async () => {
      if (!form) return
      const { data } = await getApi().put<{ data: EmailSettings }>('/settings/email', {
        smtp_host: form.smtp_host,
        smtp_port: form.smtp_port ? Number(form.smtp_port) : undefined,
        smtp_username: form.smtp_username,
        smtp_password: form.smtp_password || undefined,
        smtp_from_email: form.smtp_from_email,
        smtp_from_name: form.smtp_from_name,
      })
      return data.data
    },
    onSuccess: (s) => {
      toast.success('Email settings saved')
      syncForm(s)
      queryClient.invalidateQueries({ queryKey: ['email-settings'] })
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  const testMutation = useMutation({
    mutationFn: async () => {
      const { data } = await getApi().post<{ data: unknown }>('/settings/email/test', {
        to: testEmail,
      })
      return data
    },
    onSuccess: () => toast.success('Test email sent successfully'),
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex justify-center py-10">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    )
  }

  if (!form) {
    return null
  }

  const set = (key: keyof typeof form, value: string) =>
    setForm((f) => (f ? { ...f, [key]: value } : f))

  return (
    <Card>
      <CardHeader>
        <CardTitle>SMTP configuration</CardTitle>
        <CardDescription>
          Used to send alert emails. You can test before saving.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="host">SMTP host</Label>
            <Input
              id="host"
              value={form.smtp_host}
              onChange={(e) => set('smtp_host', e.target.value)}
              placeholder="smtp.example.com"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="port">Port</Label>
            <Input
              id="port"
              type="number"
              value={form.smtp_port}
              onChange={(e) => set('smtp_port', e.target.value)}
              placeholder="587"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="user">Username</Label>
            <Input
              id="user"
              value={form.smtp_username}
              onChange={(e) => set('smtp_username', e.target.value)}
              autoComplete="off"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="pass">Password</Label>
            <Input
              id="pass"
              type="password"
              value={form.smtp_password}
              onChange={(e) => set('smtp_password', e.target.value)}
              placeholder="Leave blank to keep current"
              autoComplete="new-password"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="from">From email</Label>
            <Input
              id="from"
              type="email"
              value={form.smtp_from_email}
              onChange={(e) => set('smtp_from_email', e.target.value)}
              placeholder="watchtower@example.com"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="fromName">From name</Label>
            <Input
              id="fromName"
              value={form.smtp_from_name}
              onChange={(e) => set('smtp_from_name', e.target.value)}
              placeholder="WatchTower"
            />
          </div>
        </div>

        <div className="flex items-end gap-2 border-t border-border pt-4">
          <div className="flex-1 space-y-2">
            <Label htmlFor="test-email">Send test email to</Label>
            <Input
              id="test-email"
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
          <Button onClick={() => updateMutation.mutate()} disabled={updateMutation.isPending}>
            {updateMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            Save
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function AlertSettingsTab() {
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['alert-settings'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: AlertSettings }>('/settings/alert')
      return data.data
    },
  })

  const [window, setWindow] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: async () => {
      const { data } = await getApi().put<{ data: AlertSettings }>('/settings/alert', {
        throttle_window: window ? Number(window) : undefined,
      })
      return data.data
    },
    onSuccess: (s) => {
      toast.success('Alert settings saved')
      setWindow(String(s.throttle_window))
      queryClient.invalidateQueries({ queryKey: ['alert-settings'] })
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex justify-center py-10">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    )
  }

  const value = window ?? (data ? String(data.throttle_window) : '60')

  return (
    <Card>
      <CardHeader>
        <CardTitle>Alert throttling</CardTitle>
        <CardDescription>
          Don&apos;t re-alert on the same issue within this window.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="max-w-xs space-y-2">
          <Label htmlFor="throttle">Throttle window (minutes)</Label>
          <Input
            id="throttle"
            type="number"
            min={1}
            value={value}
            onChange={(e) => setWindow(e.target.value)}
          />
        </div>
        <Button
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending}
        >
          {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
          Save
        </Button>
      </CardContent>
    </Card>
  )
}
