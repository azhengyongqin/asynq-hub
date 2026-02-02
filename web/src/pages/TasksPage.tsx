import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { TableHeader, TableBody, TableHead, TableRow, TableCell } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'
import { 
  RefreshCw, 
  Search, 
  Play, 
  Clock, 
  AlertCircle, 
  CheckCircle2, 
  XCircle, 
  Activity, 
  ChevronRight, 
  Hash, 
  Calendar, 
  Layers, 
  ChevronLeft, 
  Copy, 
  Check,
  Filter,
  X,
  Info,
  Server,
  Cpu
} from 'lucide-react'
import { cn } from '@/lib/utils'

type Task = {
  task_id: string
  worker_name: string
  queue: string
  priority: number
  payload: any
  status: string
  last_attempt: number
  last_error?: string
  last_worker_name?: string
  trace_id?: string
  created_at: string
  updated_at: string
}

type Attempt = {
  task_id: string
  asynq_task_id?: string
  attempt: number
  status: string
  started_at: string
  finished_at?: string
  duration_ms?: number
  error?: string
  worker_name?: string
  trace_id?: string
  span_id?: string
}

const STATUS_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  pending: "secondary",
  running: "default",
  success: "outline",
  fail: "destructive",
  dead: "destructive",
}

const STATUS_ICONS: Record<string, any> = {
  pending: Clock,
  running: Play,
  success: CheckCircle2,
  fail: AlertCircle,
  dead: XCircle,
}

export default function TasksPage() {
  const { t } = useTranslation()

  // Helper to format simplified relative time
  const timeAgo = (dateStr: string) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffSec = Math.floor(diffMs / 1000)
    const diffMin = Math.floor(diffSec / 60)
    const diffHour = Math.floor(diffMin / 60)
    const diffDay = Math.floor(diffHour / 24)

    if (diffSec < 60) return t('tasks.time.justNow')
    if (diffMin < 60) return t('tasks.time.minutesAgo').replace('{0}', diffMin.toString())
    if (diffHour < 24) return t('tasks.time.hoursAgo').replace('{0}', diffHour.toString())
    return t('tasks.time.daysAgo').replace('{0}', diffDay.toString())
  }

  const [items, setItems] = useState<Task[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const [filters, setFilters] = useState({ worker_name: '', status: '', queue: '', task_id: '' })
  const [workers, setWorkers] = useState<string[]>([])

  const ALL = '__all__'
  const statusOptions = useMemo(() => ['', 'pending', 'running', 'success', 'fail', 'dead'] as const, [])
  const [queueGroups, setQueueGroups] = useState<string[]>([])

  const [detailOpen, setDetailOpen] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailError, setDetailError] = useState<string | null>(null)
  const [selected, setSelected] = useState<{ task?: Task; attempts?: Attempt[] } | null>(null)
  const [copied, setCopied] = useState<string | null>(null)

  const queryString = useMemo(() => {
    const p = new URLSearchParams()
    if (filters.worker_name.trim()) p.set('worker_name', filters.worker_name.trim())
    if (filters.status.trim()) p.set('status', filters.status.trim())
    if (filters.queue.trim()) p.set('queue', filters.queue.trim())
    if (filters.task_id.trim()) p.set('task_id', filters.task_id.trim())
    
    // Pagination params
    p.set('limit', pageSize.toString())
    p.set('offset', ((page - 1) * pageSize).toString())
    
    return p.toString()
  }, [filters, page, pageSize])

  async function refresh() {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`/api/v1/tasks${queryString ? `?${queryString}` : ''}`)
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { error?: string } | null
        throw new Error(data?.error || `${t('workers.requestFailed')}: ${res.status}`)
      }
      const data = (await res.json()) as { items: Task[], total: number }
      setItems(data.items ?? [])
      setTotal(data.total || 0)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('workers.unknownError'))
    } finally {
      setLoading(false)
    }
  }

  async function openDetail(task: Task) {
    setSelected({ task })
    setDetailOpen(true)
    setDetailLoading(true)
    setDetailError(null)
    try {
      const res = await fetch(`/api/v1/tasks/${encodeURIComponent(task.task_id)}`)
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { error?: string } | null
        throw new Error(data?.error || `${t('workers.requestFailed')}: ${res.status}`)
      }
      const data = (await res.json()) as { item: Task; attempts: Attempt[] }
      setSelected({ task: data.item, attempts: data.attempts })
    } catch (e) {
      setDetailError(e instanceof Error ? e.message : t('workers.unknownError'))
    } finally {
      setDetailLoading(false)
    }
  }

  async function replay(taskId: string) {
    setDetailLoading(true)
    setDetailError(null)
    try {
      const res = await fetch(`/api/v1/tasks/${encodeURIComponent(taskId)}/replay`, { method: 'POST' })
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { error?: string } | null
        throw new Error(data?.error || `${t('workers.requestFailed')}: ${res.status}`)
      }
      await refresh()
      if (selected?.task) await openDetail(selected.task)
    } catch (e) {
      setDetailError(e instanceof Error ? e.message : t('workers.unknownError'))
    } finally {
      setDetailLoading(false)
    }
  }

  async function loadWorkers() {
    try {
      const res = await fetch('/api/v1/workers')
      if (!res.ok) return
      const data = (await res.json()) as { 
        items: { 
          worker_name: string
          queue_groups?: { name: string }[]
        }[] 
      }
      const items = data.items ?? []
      setWorkers(items.map((x) => x.worker_name))
      
      // 提取所有唯一的队列组名称
      const allQueueGroups = new Set<string>()
      items.forEach((worker) => {
        (worker.queue_groups || []).forEach((qg) => {
          allQueueGroups.add(qg.name)
        })
      })
      setQueueGroups(Array.from(allQueueGroups).sort())
    } catch {
      // Ignore
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    setCopied(text)
    setTimeout(() => setCopied(null), 2000)
  }

  useEffect(() => {
    refresh()
    loadWorkers()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, filters])

  useEffect(() => {
     setPage(1)
  }, [filters])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const isFiltered = filters.worker_name !== '' || filters.status !== '' || filters.queue !== '' || filters.task_id !== ''

  return (
    <div className="flex flex-col h-full gap-4 min-h-0">
      {/* Search & Filters Card */}
      <Card className="shadow-sm border-none bg-muted/20 flex-none">
        <CardContent className="p-3">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
            <div className="relative flex-1 group">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
              <Input 
                placeholder={t('tasks.filters.searchTaskId', 'Search Task ID...')} 
                className="pl-9 h-9 border-none bg-background/50 focus-visible:bg-background transition-all"
                value={filters.task_id}
                onChange={(e) => setFilters(p => ({ ...p, task_id: e.target.value }))}
              />
              {filters.task_id && (
                <button 
                  onClick={() => setFilters(p => ({ ...p, task_id: '' }))}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </div>
            
            <div className="flex flex-wrap items-center gap-2">
              <div className="flex items-center gap-2 bg-background/50 p-1 rounded-md border border-transparent">
                <Filter className="h-3.5 w-3.5 text-muted-foreground ml-1" />
                <Select
                  value={filters.worker_name === '' ? ALL : filters.worker_name}
                  onValueChange={(v) => setFilters((p) => ({ ...p, worker_name: v === ALL ? '' : v }))}
                >
                  <SelectTrigger className="w-[140px] h-7 border-none bg-transparent text-xs font-medium focus:ring-0">
                    <SelectValue placeholder={t('tasks.filters.workerName')} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={ALL}>{t('tasks.filters.allWorkers')}</SelectItem>
                    {workers.map((w) => (
                      <SelectItem key={w} value={w}>{w}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>

                <div className="w-px h-4 bg-border/50" />

                <Select
                  value={filters.status === '' ? ALL : filters.status}
                  onValueChange={(v) => setFilters((p) => ({ ...p, status: v === ALL ? '' : v }))}
                >
                  <SelectTrigger className="w-[110px] h-7 border-none bg-transparent text-xs font-medium focus:ring-0">
                    <SelectValue placeholder={t('tasks.filters.status')} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={ALL}>{t('tasks.filters.allStatus')}</SelectItem>
                    {statusOptions.filter((x) => x !== '').map((s) => (
                      <SelectItem key={s} value={s}>{s}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>

                <div className="w-px h-4 bg-border/50" />

                <Select
                  value={filters.queue === '' ? ALL : filters.queue}
                  onValueChange={(v) => setFilters((p) => ({ ...p, queue: v === ALL ? '' : v }))}
                >
                  <SelectTrigger className="w-[130px] h-7 border-none bg-transparent text-xs font-medium focus:ring-0">
                    <SelectValue placeholder={t('tasks.filters.queue')} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={ALL}>{t('tasks.filters.allQueues')}</SelectItem>
                    {queueGroups.map((q) => (
                      <SelectItem key={q} value={q}>{q}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {isFiltered && (
                <Button 
                  variant="ghost" 
                  size="sm" 
                  className="h-8 text-xs text-muted-foreground hover:text-foreground"
                  onClick={() => setFilters({ worker_name: '', status: '', queue: '', task_id: '' })}
                >
                  {t('tasks.filters.reset')}
                </Button>
              )}

              <div className="flex items-center gap-2 ml-auto lg:ml-2">
                <Button variant="default" size="sm" onClick={refresh} disabled={loading} className="h-8 shadow-sm">
                  <RefreshCw className={cn("mr-2 h-3.5 w-3.5", loading && "animate-spin")} />
                  {t('tasks.actions.refresh')}
                </Button>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {error && (
        <div className="rounded-lg bg-destructive/10 border border-destructive/20 p-3 text-sm text-destructive flex items-center gap-2 flex-none">
          <AlertCircle className="h-4 w-4" />
          <p><span className="font-bold">Error:</span> {error}</p>
        </div>
      )}

      {/* Main Table Card */}
      <Card className="shadow-md border-none flex-1 overflow-hidden flex flex-col min-h-0 bg-background/50 backdrop-blur-sm">
        <div className="flex-1 overflow-auto relative scrollbar-thin scrollbar-thumb-muted-foreground/20">
          <table className="w-full caption-bottom text-sm border-collapse">
            <TableHeader className="sticky top-0 z-20 bg-background/95 backdrop-blur-md border-b">
              <TableRow className="hover:bg-transparent">
                <TableHead className="w-[20%] pl-6 h-12 text-[10px] font-bold uppercase tracking-widest text-muted-foreground">{t('tasks.table.taskId')}</TableHead>
                <TableHead className="w-[15%] h-12 text-[10px] font-bold uppercase tracking-widest text-muted-foreground">{t('tasks.table.worker')}</TableHead>
                <TableHead className="w-[12%] h-12 text-[10px] font-bold uppercase tracking-widest text-muted-foreground">{t('tasks.table.queue')}</TableHead>
                <TableHead className="w-[10%] h-12 text-[10px] font-bold uppercase tracking-widest text-muted-foreground">{t('tasks.table.status')}</TableHead>
                <TableHead className="w-[15%] h-12 text-[10px] font-bold uppercase tracking-widest text-muted-foreground">{t('tasks.table.execution')}</TableHead>
                <TableHead className="w-[13%] h-12 text-[10px] font-bold uppercase tracking-widest text-muted-foreground">{t('tasks.table.lastWorker')}</TableHead>
                <TableHead className="w-[15%] text-right pr-6 h-12 text-[10px] font-bold uppercase tracking-widest text-muted-foreground">{t('tasks.table.updated')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading && items.length === 0 ? (
                 Array.from({ length: 10 }).map((_, i) => (
                   <TableRow key={i} className="border-b border-border/40">
                     <TableCell className="py-4 pl-6"><Skeleton className="h-5 w-3/4" /></TableCell>
                     <TableCell><Skeleton className="h-5 w-24" /></TableCell>
                     <TableCell><Skeleton className="h-5 w-20" /></TableCell>
                     <TableCell><Skeleton className="h-5 w-20" /></TableCell>
                     <TableCell><Skeleton className="h-5 w-full" /></TableCell>
                     <TableCell><Skeleton className="h-5 w-24" /></TableCell>
                     <TableCell className="pr-6"><div className="flex justify-end"><Skeleton className="h-5 w-16" /></div></TableCell>
                   </TableRow>
                 ))
              ) : (
                items.map((t) => {
                  const StatusIcon = STATUS_ICONS[t.status] || AlertCircle
                  const isSuccess = t.status === 'success'
                  const isFail = t.status === 'fail' || t.status === 'dead'
                  
                  return (
                    <TableRow 
                      key={t.task_id} 
                      className="cursor-pointer group hover:bg-primary/[0.03] transition-all duration-200 border-b border-border/40 last:border-0"
                      onClick={() => openDetail(t)}
                    >
                      <TableCell className="py-4 pl-6 align-middle">
                        <div className="flex items-center gap-2">
                           <div className="font-mono text-[11px] font-semibold text-foreground/80 group-hover:text-primary transition-colors truncate max-w-[200px]" title={t.task_id}>
                             {t.task_id}
                           </div>
                           <Button
                             variant="ghost" 
                             size="icon" 
                             className="h-6 w-6 rounded-full opacity-0 group-hover:opacity-100 transition-opacity bg-background border"
                             onClick={(e) => {
                               e.stopPropagation()
                               copyToClipboard(t.task_id)
                             }}
                           >
                             {copied === t.task_id ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
                           </Button>
                        </div>
                      </TableCell>

                      <TableCell className="py-4 align-middle">
                        <div className="flex items-center gap-2">
                          <Server className="h-3.5 w-3.5 text-muted-foreground/60" />
                          <span className="text-xs font-bold text-foreground/70 tracking-tighter uppercase">{t.worker_name}</span>
                        </div>
                      </TableCell>

                      <TableCell className="py-4 align-middle">
                        <div className="text-xs font-medium text-muted-foreground/80">
                          {t.queue}
                        </div>
                      </TableCell>
                      
                      <TableCell className="py-4 align-middle">
                        <Badge 
                          variant={STATUS_VARIANTS[t.status] || 'outline'}
                          className={cn(
                            "capitalize gap-1.5 py-0.5 px-2.5 shadow-sm border font-bold text-[10px] rounded-full", 
                            isSuccess && "border-green-200 text-green-700 bg-green-50 dark:bg-green-900/20 dark:text-green-400 dark:border-green-800",
                            isFail && "border-red-200 text-red-700 bg-red-50 dark:bg-red-900/20 dark:text-red-400 dark:border-red-800",
                            t.status === 'pending' && "border-slate-200 bg-slate-50 text-slate-600 dark:bg-slate-800/20 dark:text-slate-400 dark:border-slate-700",
                            t.status === 'running' && "border-blue-200 bg-blue-50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-800 animate-pulse"
                          )}
                        >
                          <StatusIcon className="h-3 w-3" />
                          {t.status}
                        </Badge>
                      </TableCell>

                      <TableCell className="py-4 align-middle">
                        <div className="flex items-center gap-3 text-xs text-muted-foreground">
                           <div className="flex items-center gap-1.5 px-1.5 py-0.5 rounded bg-muted/40 font-mono" title="Priority">
                             <Layers className="h-3 w-3 opacity-60" />
                             <span className="font-bold">{t.priority}</span>
                           </div>
                           <div className="flex items-center gap-1.5 px-1.5 py-0.5 rounded bg-muted/40 font-mono" title="Attempts">
                             <Activity className="h-3 w-3 opacity-60" />
                             <span className="font-bold">{t.last_attempt}</span>
                           </div>
                        </div>
                      </TableCell>

                      <TableCell className="py-4 align-middle">
                        <div className="text-xs font-mono text-muted-foreground truncate max-w-[120px] bg-muted/30 px-1.5 py-0.5 rounded" title={t.last_worker_name || 'N/A'}>
                          {t.last_worker_name || '-'}
                        </div>
                      </TableCell>

                      <TableCell className="py-4 pr-6 align-middle text-right">
                        <div className="flex flex-col items-end">
                          <span className="text-xs font-bold text-foreground/80 whitespace-nowrap">{timeAgo(t.updated_at)}</span>
                          <span className="text-[10px] font-medium text-muted-foreground/60">
                            {new Date(t.updated_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                          </span>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
              {!loading && items.length === 0 && (
                <TableRow className="hover:bg-transparent">
                  <TableCell colSpan={7} className="h-64 text-center">
                    <div className="flex flex-col items-center justify-center text-muted-foreground space-y-4">
                       <div className="h-20 w-20 rounded-full bg-muted/30 flex items-center justify-center animate-in zoom-in-50 duration-300">
                         <Search className="h-10 w-10 opacity-20" />
                       </div>
                       <div className="space-y-1">
                        <p className="font-bold text-lg text-foreground/80">{t('tasks.empty.title')}</p>
                        <p className="text-sm text-muted-foreground/70 max-w-xs mx-auto">{t('tasks.empty.hint')}</p>
                       </div>
                       {isFiltered && (
                         <Button variant="outline" size="sm" onClick={() => setFilters({ worker_name: '', status: '', queue: '', task_id: '' })}>
                           Clear All Filters
                         </Button>
                       )}
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </table>
        </div>
        
        {/* Pagination Footer */}
        <div className="flex items-center justify-between p-4 bg-muted/20 border-t flex-none">
           <div className="flex items-center gap-4">
              <div className="text-[11px] font-medium text-muted-foreground">
                Showing <span className="font-bold text-foreground">{items.length}</span> of <span className="font-bold text-foreground">{total}</span> tasks
              </div>
              <div className="flex items-center gap-2">
                 <span className="text-[11px] font-bold text-muted-foreground uppercase tracking-tighter">Rows</span>
                 <Select value={pageSize.toString()} onValueChange={(v) => setPageSize(Number(v))}>
                   <SelectTrigger className="w-[65px] h-7 text-[11px] font-bold bg-background border-none shadow-sm">
                     <SelectValue />
                   </SelectTrigger>
                   <SelectContent>
                     <SelectItem value="20">20</SelectItem>
                     <SelectItem value="50">50</SelectItem>
                     <SelectItem value="100">100</SelectItem>
                   </SelectContent>
                 </Select>
              </div>
           </div>
           
           <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page === 1 || loading}
                className="h-7 px-3 text-[11px] font-bold uppercase tracking-tighter bg-background shadow-sm hover:bg-muted"
              >
                <ChevronLeft className="h-3.5 w-3.5 mr-1" />
                {t('tasks.pagination.prev')}
              </Button>
              <div className="text-[11px] font-bold mx-2 min-w-[3rem] text-center text-muted-foreground">
                 {page} <span className="mx-1 opacity-30">/</span> {totalPages}
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                disabled={page === totalPages || loading}
                className="h-7 px-3 text-[11px] font-bold uppercase tracking-tighter bg-background shadow-sm hover:bg-muted"
              >
                {t('tasks.pagination.next')}
                <ChevronRight className="h-3.5 w-3.5 ml-1" />
              </Button>
           </div>
        </div>
      </Card>

      {/* Detail Sheet */}
      <Sheet open={detailOpen} onOpenChange={setDetailOpen}>
        <SheetContent className="sm:max-w-2xl w-full flex flex-col p-0 gap-0 overflow-hidden">
          <SheetHeader className="p-6 border-b bg-muted/10">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2 text-[10px] font-black tracking-widest text-primary uppercase">
                <Info className="h-4 w-4" />
                {t('tasks.detail.title')}
              </div>
              <Button 
                size="sm" 
                variant="outline" 
                onClick={() => replay(selected?.task?.task_id || '')} 
                disabled={detailLoading} 
                className="h-8 text-[11px] font-bold uppercase tracking-widest bg-primary text-primary-foreground hover:bg-primary/90 border-none shadow-md"
              >
                <RefreshCw className={cn("mr-2 h-3 w-3", detailLoading && "animate-spin")} />
                {t('tasks.detail.replayTask')}
              </Button>
            </div>
            
            <div className="flex items-center gap-3">
              <SheetTitle className="text-xl font-black tracking-tight font-mono truncate max-w-[400px]">
                {selected?.task?.task_id}
              </SheetTitle>
              <Button 
                variant="ghost" 
                size="icon" 
                className="h-8 w-8 rounded-full hover:bg-muted" 
                onClick={() => selected?.task && copyToClipboard(selected.task.task_id)}
              >
                {copied === selected?.task?.task_id ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4 text-muted-foreground" />}
              </Button>
            </div>
          </SheetHeader>

          <div className="flex-1 overflow-y-auto p-6 space-y-8 scrollbar-thin scrollbar-thumb-muted-foreground/20">
            {detailError && (
              <div className="text-sm text-destructive bg-destructive/10 p-4 rounded-xl border border-destructive/20 flex items-center gap-3 animate-in fade-in duration-300">
                <AlertCircle className="h-5 w-5 flex-none" />
                <p>{detailError}</p>
              </div>
            )}
            
            {selected?.task ? (
              <div className="space-y-8 animate-in fade-in slide-in-from-right-4 duration-500">
                {/* Information Grid */}
                <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                  {[
                    { label: t('tasks.detail.workerName'), value: selected.task.worker_name, icon: Server },
                    { 
                      label: t('tasks.detail.status'), 
                      value: selected.task.status, 
                      icon: STATUS_ICONS[selected.task.status] || Info,
                      className: cn(
                        "font-black uppercase tracking-widest",
                        selected.task.status === 'success' && "text-green-500",
                        (selected.task.status === 'fail' || selected.task.status === 'dead') && "text-red-500",
                        selected.task.status === 'running' && "text-blue-500",
                        selected.task.status === 'pending' && "text-slate-500"
                      )
                    },
                    { label: t('tasks.detail.queue'), value: selected.task.queue, icon: Layers },
                    { label: t('tasks.detail.attempts'), value: selected.task.last_attempt, icon: Activity },
                    { label: t('tasks.detail.priority'), value: selected.task.priority, icon: Hash },
                    { label: t('tasks.detail.lastWorkerInstance'), value: selected.task.last_worker_name || '-', icon: Cpu },
                  ].map((item, i) => (
                    <div key={i} className="p-4 rounded-2xl bg-muted/30 border border-transparent hover:border-border transition-colors group">
                      <div className="flex items-center gap-2 mb-2">
                        <item.icon className="h-3.5 w-3.5 text-muted-foreground group-hover:text-primary transition-colors" />
                        <span className="text-[10px] text-muted-foreground uppercase font-black tracking-widest">{item.label}</span>
                      </div>
                      <div className={cn("font-bold text-sm truncate", item.className)}>
                        {item.value}
                      </div>
                    </div>
                  ))}
                </div>

                {/* Error Box */}
                {(selected.task.last_error || selected.task.status === 'fail' || selected.task.status === 'dead') && (
                  <div className="rounded-2xl border border-red-200 bg-red-50/50 dark:bg-red-900/10 p-5 space-y-3 shadow-sm">
                    <div className="flex items-center gap-2 text-red-600 dark:text-red-400 font-black text-[10px] uppercase tracking-widest">
                      <AlertCircle className="h-4 w-4" />
                      {t('tasks.detail.lastError')}
                    </div>
                    <div className="text-xs font-mono text-red-700 dark:text-red-300 break-all leading-relaxed bg-background/50 p-3 rounded-lg border border-red-100 dark:border-red-900/30">
                      {selected.task.last_error || t('tasks.detail.noError')}
                    </div>
                  </div>
                )}

                {/* Payload Area */}
                <div className="space-y-4">
                   <div className="flex items-center justify-between">
                     <h3 className="text-xs font-black uppercase tracking-widest text-muted-foreground flex items-center gap-2">
                       <Layers className="h-4 w-4 text-primary" />
                       {t('tasks.detail.taskPayload')}
                     </h3>
                     <Button 
                      variant="outline" 
                      size="sm" 
                      className="h-7 text-[10px] font-black uppercase tracking-widest rounded-full"
                      onClick={() => copyToClipboard(JSON.stringify(selected.task!.payload, null, 2))}
                     >
                        {copied === JSON.stringify(selected.task!.payload, null, 2) ? (
                          <><Check className="h-3 w-3 mr-1 text-green-500" /> {t('tasks.detail.copied')}</>
                        ) : (
                          <><Copy className="h-3 w-3 mr-1" /> {t('tasks.detail.copyJson')}</>
                        )}
                     </Button>
                   </div>
                   <div className="rounded-2xl border bg-muted/20 p-5 font-mono text-[11px] overflow-x-auto max-h-[400px] shadow-inner group">
                     <pre className="leading-relaxed group-hover:text-foreground transition-colors">
                       {JSON.stringify(selected.task.payload, null, 2)}
                     </pre>
                   </div>
                </div>

                {/* Timeline */}
                <div className="space-y-6">
                  <h3 className="text-xs font-black uppercase tracking-widest text-muted-foreground flex items-center gap-2">
                    <Activity className="h-4 w-4 text-primary" />
                    {t('tasks.detail.executionHistory')}
                  </h3>

                  <div className="space-y-0 relative pl-4 border-l-2 border-muted/50 ml-2">
                    {(selected.attempts ?? []).map((a, i) => (
                      <div key={i} className="mb-8 last:mb-0 relative">
                         <div className={cn("absolute -left-[21px] top-1 h-4 w-4 rounded-full border-4 border-background shadow-md transition-transform hover:scale-125", 
                            a.status === 'success' ? "bg-green-500" : (a.status === 'fail' || a.status === 'dead') ? "bg-red-500" : a.status === 'running' ? "bg-blue-500" : "bg-slate-400"
                         )} />
                         
                         <div className="flex flex-col gap-3 bg-muted/20 hover:bg-muted/40 rounded-2xl p-4 border border-transparent hover:border-border transition-all duration-300">
                            <div className="flex items-center justify-between flex-wrap gap-2">
                               <div className="flex items-center gap-3">
                                  <span className="font-black text-xs uppercase tracking-widest">{t('tasks.detail.attempt').replace('{0}', a.attempt.toString())}</span>
                                  <Badge 
                                    variant={STATUS_VARIANTS[a.status] || 'outline'}
                                    className={cn(
                                      "capitalize px-2 h-5 text-[9px] font-black border-none shadow-none",
                                      a.status === 'success' && "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300",
                                      (a.status === 'fail' || a.status === 'dead') && "bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300",
                                      a.status === 'running' && "bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300",
                                    )}
                                  >
                                    {a.status}
                                  </Badge>
                               </div>
                               <div className="text-[10px] text-muted-foreground font-black uppercase tracking-tighter flex items-center gap-1.5 opacity-60">
                                 <Calendar className="h-3 w-3" />
                                 {new Date(a.started_at).toLocaleString()}
                               </div>
                            </div>
                            
                            <div className="grid grid-cols-2 gap-y-2 text-[11px] font-medium">
                               <div className="flex items-center gap-2">
                                 <Server className="h-3 w-3 opacity-40" />
                                 <span className="text-muted-foreground">{t('tasks.detail.worker')}:</span>
                                 <span className="font-bold font-mono tracking-tighter">{a.worker_name || 'N/A'}</span>
                               </div>
                               {a.duration_ms && (
                                 <div className="flex items-center gap-2 justify-end">
                                   <Clock className="h-3 w-3 opacity-40" />
                                   <span className="text-muted-foreground">{t('tasks.detail.duration')}:</span>
                                   <span className="font-black text-orange-500">{a.duration_ms}ms</span>
                                 </div>
                               )}
                               {a.trace_id && (
                                 <div className="flex items-center gap-2 col-span-2 mt-1">
                                   <Search className="h-3 w-3 opacity-40" />
                                   <span className="text-muted-foreground font-black text-[9px] uppercase tracking-widest">{t('tasks.detail.traceId')}:</span>
                                   <span className="font-mono text-[10px] opacity-80 break-all">{a.trace_id}</span>
                                 </div>
                               )}
                            </div>

                            {a.error && (
                              <div className="mt-2 rounded-xl border border-red-100 dark:border-red-900/30 bg-red-50/20 dark:bg-red-950/20 p-3 text-[10px] text-red-600 dark:text-red-400 font-mono break-all leading-normal">
                                {a.error}
                              </div>
                            )}
                         </div>
                      </div>
                    ))}
                    {(selected.attempts ?? []).length === 0 && (
                      <div className="flex flex-col items-center justify-center py-10 bg-muted/10 rounded-3xl border-2 border-dashed border-muted">
                        <Activity className="h-8 w-8 text-muted-foreground opacity-20 mb-2" />
                        <span className="text-xs text-muted-foreground font-bold uppercase tracking-widest">{t('tasks.detail.noHistory')}</span>
                      </div>
                    )}
                  </div>
                </div>

                {/* Final Metadata */}
                <div className="pt-6 border-t flex items-center justify-between text-[10px] text-muted-foreground font-black uppercase tracking-widest opacity-50">
                  <div className="flex items-center gap-2">
                    <div className="h-1.5 w-1.5 rounded-full bg-muted-foreground" />
                    {t('tasks.detail.created')}: <span className="text-foreground ml-1">{new Date(selected.task.created_at).toLocaleString()}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    {t('tasks.detail.updated')}: <span className="text-foreground ml-1">{new Date(selected.task.updated_at).toLocaleString()}</span>
                    <div className="h-1.5 w-1.5 rounded-full bg-primary animate-pulse" />
                  </div>
                </div>
              </div>
            ) : (
              <div className="p-6 space-y-8 animate-pulse">
                <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                  {Array.from({ length: 6 }).map((_, i) => (
                    <div key={i} className="h-20 bg-muted rounded-2xl" />
                  ))}
                </div>
                <div className="h-48 bg-muted rounded-2xl" />
                <div className="h-64 bg-muted rounded-2xl" />
              </div>
            )}
          </div>
        </SheetContent>
      </Sheet>
    </div>
  )
}
