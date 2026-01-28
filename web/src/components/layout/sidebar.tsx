import { Server, ListTodo, Activity } from "lucide-react"
import { useTranslation } from "react-i18next"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"

interface SidebarProps {
  activeTab: string
  setActiveTab: (tab: string) => void
  className?: string
  onItemClick?: () => void
}

export function Sidebar({ activeTab, setActiveTab, className, onItemClick }: SidebarProps) {
  const { t } = useTranslation()

  const navItems = [
    { id: 'workers', label: t('nav.workers'), icon: Server },
    { id: 'tasks', label: t('nav.tasks'), icon: ListTodo },
  ]

  return (
    <div className={cn("flex flex-col h-full border-r bg-background", className)}>
      <div className="flex h-14 items-center border-b px-6">
        <Activity className="mr-2 h-6 w-6 text-primary" />
        <span className="font-bold text-lg tracking-tight">{t('app.title')}</span>
      </div>
      <div className="flex-1 py-6 px-4">
        <div className="space-y-2">
          <h2 className="mb-2 px-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
            {t('nav.platform')}
          </h2>
          <nav className="space-y-1">
            {navItems.map((item) => {
              const Icon = item.icon
              return (
                <Button
                  key={item.id}
                  variant={activeTab === item.id ? "secondary" : "ghost"}
                  className={cn(
                    "w-full justify-start px-3 py-2 h-9 font-medium transition-all duration-200",
                    activeTab === item.id 
                      ? "bg-primary/10 text-primary hover:bg-primary/15" 
                      : "text-muted-foreground hover:text-foreground hover:bg-muted"
                  )}
                  onClick={() => {
                    setActiveTab(item.id)
                    onItemClick?.()
                  }}
                >
                  <Icon className="mr-2 h-4 w-4" />
                  {item.label}
                </Button>
              )
            })}
          </nav>
        </div>
      </div>
      <div className="border-t p-4 text-xs text-muted-foreground bg-muted/20">
        <p className="font-medium">{t('app.subtitle')}</p>
        <p className="mt-1 opacity-70">{t('app.version')}</p>
      </div>
    </div>
  )
}
