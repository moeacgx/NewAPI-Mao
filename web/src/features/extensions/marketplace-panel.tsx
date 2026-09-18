import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Download, RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Empty, EmptyTitle } from '@/components/ui/empty'

import {
  getExtensionMarketplaceConfig,
  uploadMarketplaceExtension,
} from './api'
import {
  downloadExtensionArchive,
  fetchExtensionCatalog,
  type ExtensionCatalogModule,
} from './marketplace-download'
import { getExtensionQueryKey } from './query-key'

export function ExtensionMarketplacePanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selected, setSelected] = useState<ExtensionCatalogModule | null>(null)
  const catalog = useQuery({
    queryKey: ['extension-marketplace'],
    queryFn: async () => {
      const config = await getExtensionMarketplaceConfig()
      return {
        config,
        catalog: await fetchExtensionCatalog(config.catalog_url),
      }
    },
    staleTime: 60000,
    retry: false,
  })
  const install = useMutation({
    mutationFn: async (entry: ExtensionCatalogModule) => {
      if (!catalog.data) throw new Error('Failed to load online modules')
      const config = catalog.data.config
      const file = await downloadExtensionArchive(
        config.catalog_url,
        entry,
        config.max_archive_bytes
      )
      return uploadMarketplaceExtension(file, entry, config.catalog_url)
    },
    onSuccess: async () => {
      setSelected(null)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: getExtensionQueryKey() }),
        queryClient.invalidateQueries({ queryKey: ['extensions', 'admin'] }),
      ])
      toast.success(t('Online module installed'))
    },
  })

  return (
    <Card>
      <CardHeader className='flex flex-wrap items-start justify-between gap-3 sm:flex-row'>
        <div className='min-w-0 space-y-1'>
          <CardTitle>{t('Online modules')}</CardTitle>
          <CardDescription>
            {t(
              'Install a specific module version from the extension repository.'
            )}
          </CardDescription>
          {catalog.data && (
            <p className='text-muted-foreground text-xs'>
              {t('Current host: {{version}}', {
                version: catalog.data.config.host_version,
              })}
            </p>
          )}
        </div>
        <Button
          variant='outline'
          size='sm'
          disabled={catalog.isFetching || install.isPending}
          onClick={() => void catalog.refetch()}
        >
          <RefreshCw className='size-4' />
          {t('Refresh catalog')}
        </Button>
      </CardHeader>
      <CardContent className='space-y-3'>
        {catalog.isPending && (
          <p role='status' className='text-muted-foreground py-6 text-sm'>
            {t('Loading online modules...')}
          </p>
        )}
        {catalog.error && (
          <Alert variant='destructive'>
            <AlertTitle>{t('Failed to load online modules')}</AlertTitle>
            <AlertDescription className='flex flex-wrap items-center justify-between gap-2'>
              <span className='break-all'>{t(catalog.error.message)}</span>
              <Button
                variant='outline'
                size='sm'
                disabled={catalog.isFetching}
                onClick={() => void catalog.refetch()}
              >
                {t('Retry')}
              </Button>
            </AlertDescription>
          </Alert>
        )}
        {!catalog.error && catalog.data?.catalog.modules.length === 0 && (
          <Empty className='border'>
            <EmptyTitle>{t('No online modules available')}</EmptyTitle>
          </Empty>
        )}
        {!catalog.error &&
          catalog.data?.catalog.modules.map((entry) => {
            let status = entry.status || t('Not specified')
            if (entry.status === 'builtin-snapshot') {
              status = t('Built-in snapshot')
            }
            if (entry.status === 'requires-host-upgrade') {
              status = t('Host upgrade required')
            }
            if (entry.status === 'development') {
              status = t('Development version')
            }
            return (
              <div
                key={`${entry.id}@${entry.version}`}
                className='bg-card flex min-w-0 flex-col gap-3 rounded-lg border p-4 sm:flex-row sm:items-start sm:justify-between'
              >
                <div className='min-w-0 space-y-2'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <h3 className='font-medium break-all'>{entry.name}</h3>
                    <Badge variant='outline'>{entry.version}</Badge>
                    <Badge variant='secondary'>{status}</Badge>
                  </div>
                  {entry.description && (
                    <p className='text-muted-foreground text-sm break-words'>
                      {entry.description}
                    </p>
                  )}
                  <p className='text-muted-foreground text-xs break-all'>
                    {t('Host requirement: {{min}} to {{max}}', {
                      min: entry.host.min || t('Not specified'),
                      max: entry.host.max || t('No upper limit'),
                    })}
                  </p>
                  {entry.compatibility && (
                    <p className='text-sm break-words'>{entry.compatibility}</p>
                  )}
                  {entry.verification && (
                    <p className='text-muted-foreground text-xs break-words'>
                      {entry.verification}
                    </p>
                  )}
                </div>
                <Button
                  className='shrink-0'
                  disabled={install.isPending || catalog.isFetching}
                  aria-label={t('Install {{name}} {{version}}', {
                    name: entry.name,
                    version: entry.version,
                  })}
                  onClick={() => {
                    install.reset()
                    setSelected(entry)
                  }}
                >
                  <Download className='size-4' />
                  {t('Install')}
                </Button>
              </div>
            )
          })}
      </CardContent>
      <ConfirmDialog
        open={Boolean(selected)}
        onOpenChange={(open) => {
          if (!open && !install.isPending) {
            setSelected(null)
            install.reset()
          }
        }}
        title={t('Install module version')}
        desc={
          <div className='space-y-2 break-words'>
            <p>
              {t('Install {{name}} version {{version}}?', {
                name: selected?.name,
                version: selected?.version,
              })}
            </p>
            {selected?.compatibility && <p>{selected.compatibility}</p>}
            {install.error && (
              <Alert variant='destructive'>
                <AlertDescription>{t(install.error.message)}</AlertDescription>
              </Alert>
            )}
          </div>
        }
        confirmText={t('Confirm installation')}
        isLoading={install.isPending}
        handleConfirm={() => {
          if (selected && !install.isPending) install.mutate(selected)
        }}
      />
    </Card>
  )
}
