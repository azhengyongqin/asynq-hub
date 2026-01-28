import { useTranslation as useI18nTranslation } from 'react-i18next'

export function useWorkersTranslation() {
  const { t } = useI18nTranslation()
  
  return {
    t,
    translate: (key: string) => t(`workers.${key}`),
    translateStatus: (status: string) => t(`workers.status.${status}`),
    translateConfig: (key: string) => t(`workers.config.${key}`),
    translateStat: (key: string) => t(`workers.stats.${key}`),
  }
}
