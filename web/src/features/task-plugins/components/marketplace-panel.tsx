import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ErrorState } from '@/components/error-state'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Textarea } from '@/components/ui/textarea'

import {
  installMarketplacePlugin,
  listMarketplaceSources,
  listTaskPlugins,
} from '../api'
import {
  findMarketplaceVersion,
  isDefaultMarketplaceSource,
  MAOLAO_MARKETPLACE_INDEX_URL,
} from '../lib/marketplace'
import {
  downloadMarketplaceSource,
  fetchMarketplaceIndex,
} from '../lib/marketplace-download'
import type {
  MarketplaceIndexVersion,
  MarketplacePlugin,
  MarketplaceSource,
} from '../types'
import { MarketplaceSourcesDialog } from './marketplace-sources-dialog'
import { PluginVersionSelect } from './plugin-version-select'

type InstallTarget = {
  source: MarketplaceSource
  plugin: MarketplacePlugin
  version: MarketplaceIndexVersion
}

export function MarketplacePanel(props: { canManage: boolean }) {
  const { t } = useTranslation()
  const [sourcesOpen, setSourcesOpen] = useState(false)
  const [selectedUrl, setSelectedUrl] = useState('')
  const [target, setTarget] = useState<InstallTarget | null>(null)
  const [selectedVersions, setSelectedVersions] = useState<
    Record<string, string>
  >({})
  const sourcesQuery = useQuery({
    queryKey: ['task-plugin-marketplace-sources'],
    queryFn: listMarketplaceSources,
  })
  const installed = useQuery({
    queryKey: ['task-plugins'],
    queryFn: listTaskPlugins,
  })
  const sources = sourcesQuery.data ?? []
  const selected =
    sources.find((source) => source.index_url === selectedUrl) ?? sources[0]
  const index = useQuery({
    queryKey: ['task-plugin-marketplace', selected?.index_url],
    enabled: Boolean(selected),
    retry: false,
    queryFn: () => {
      if (!selected) throw new Error('No marketplace source selected')
      return fetchMarketplaceIndex(selected.index_url)
    },
  })
  return (
    <>
      <div className='flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto'>
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Stable plugins can be installed from a source. Use upload for temporary tests.'
            )}
          </p>
          {props.canManage && (
            <Button variant='outline' onClick={() => setSourcesOpen(true)}>
              {t('Manage sources')}
            </Button>
          )}
        </div>
        {sourcesQuery.isError && (
          <ErrorState
            description={t('Could not load marketplace sources')}
            onRetry={() => void sourcesQuery.refetch()}
          />
        )}
        {sourcesQuery.isPending && <p>{t('Loading...')}</p>}
        {sourcesQuery.isSuccess && sources.length === 0 && (
          <p>{t('No marketplace sources configured.')}</p>
        )}
        <div
          className='flex flex-wrap gap-2'
          aria-label={t('Marketplace sources')}
        >
          {sources.map((source) => (
            <Button
              key={source.index_url}
              variant={
                selected?.index_url === source.index_url ? 'default' : 'outline'
              }
              onClick={() => setSelectedUrl(source.index_url)}
            >
              {source.name}
            </Button>
          ))}
        </div>
        {selected && (
          <section
            className='bg-card space-y-4 rounded-xl border p-4'
            aria-label={selected.name}
          >
            <div className='flex flex-wrap items-center gap-2'>
              <h3 className='font-semibold'>{selected.name}</h3>
              {isDefaultMarketplaceSource(selected.index_url) && (
                <Badge variant='secondary'>{t('Official')}</Badge>
              )}
              {selected.index_url === MAOLAO_MARKETPLACE_INDEX_URL && (
                <Badge variant='secondary'>MaoLao</Badge>
              )}
              <Button
                variant='outline'
                size='sm'
                disabled={index.isFetching}
                onClick={() => void index.refetch()}
              >
                {t('Refresh')}
              </Button>
            </div>
            <p className='text-muted-foreground text-xs break-all'>
              {selected.index_url}
            </p>
            {index.isPending && <p>{t('Loading...')}</p>}
            {index.isError && (
              <ErrorState
                description={t('Could not load this source')}
                onRetry={() => void index.refetch()}
              />
            )}
            {index.data?.plugins.length === 0 && (
              <p>{t('This source lists no installable task plugins.')}</p>
            )}
            <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-3'>
              {index.data?.plugins.map((plugin) => {
                const versionKey = `${selected.index_url}|${plugin.key}`
                const version =
                  findMarketplaceVersion(
                    plugin,
                    selectedVersions[versionKey] ?? plugin.latest
                  ) ?? findMarketplaceVersion(plugin, plugin.latest)
                const existing = installed.data?.find(
                  (item) => item.meta.key === plugin.key
                )
                const compatible =
                  version &&
                  (version.minApiVersion ?? 1) <= 1 &&
                  (!version.kind || version.kind === 'task')
                return (
                  <article
                    key={plugin.key}
                    className='bg-card space-y-3 rounded-lg border p-4'
                  >
                    <h4 className='font-medium'>{plugin.name}</h4>
                    <p className='text-muted-foreground text-sm break-all'>
                      {plugin.key}
                    </p>
                    <PluginVersionSelect
                      aria-label={`${plugin.name} ${t('Version')}`}
                      value={version?.version}
                      options={plugin.versions.map((item) => ({
                        value: item.version,
                        label: item.version,
                      }))}
                      onValueChange={(value) =>
                        setSelectedVersions((previous) => ({
                          ...previous,
                          [versionKey]: value,
                        }))
                      }
                    />
                    {plugin.models?.length ? (
                      <p className='text-xs break-words'>
                        {plugin.models.join(', ')}
                      </p>
                    ) : null}
                    {existing && (
                      <p className='text-muted-foreground text-xs'>
                        {t(
                          'This plugin key is already installed. Matching versions require identical source.'
                        )}
                      </p>
                    )}
                    {!compatible && (
                      <p className='text-destructive text-xs'>
                        {t('This version requires a newer plugin API')}
                      </p>
                    )}
                    {!version?.sha256?.match(/^[a-f0-9]{64}$/i) && (
                      <p className='text-destructive text-xs'>
                        {t('A valid SHA-256 is required to install')}
                      </p>
                    )}
                    {props.canManage && (
                      <Button
                        variant='outline'
                        disabled={
                          !compatible ||
                          !version?.sha256?.match(/^[a-f0-9]{64}$/i)
                        }
                        onClick={() => {
                          if (version) {
                            setTarget({ source: selected, plugin, version })
                          }
                        }}
                      >
                        {t('Review and install')}
                      </Button>
                    )}
                  </article>
                )
              })}
            </div>
          </section>
        )}
      </div>
      {props.canManage && (
        <MarketplaceSourcesDialog
          open={sourcesOpen}
          onOpenChange={setSourcesOpen}
        />
      )}
      {props.canManage && target && (
        <MarketplaceInstallDialog
          key={`${target.source.index_url}/${target.plugin.key}/${target.version.version}`}
          target={target}
          onClose={() => setTarget(null)}
        />
      )}
    </>
  )
}

function MarketplaceInstallDialog(props: {
  target: InstallTarget
  onClose: () => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const version = props.target.version
  const download = useQuery({
    queryKey: [
      'task-plugin-marketplace-source',
      props.target.source.index_url,
      version.path,
      version.sha256,
    ],
    retry: false,
    queryFn: () =>
      downloadMarketplaceSource(
        props.target.source.index_url,
        version.path,
        version.sha256
      ),
  })
  const install = useMutation({
    mutationFn: async () => {
      if (!download.data) throw new Error('Plugin source is unavailable')
      return installMarketplacePlugin({
        ...download.data,
        expectedKey: props.target.plugin.key,
        expectedVersion: version.version,
        remark: props.target.source.name,
        marketplace: { ...props.target.source, path: version.path },
      })
    },
    onSuccess: () => {
      toast.success(
        t('Plugin installed disabled. Activate it from Installed plugins.')
      )
      for (const key of [
        'task-plugins',
        'task-plugin-options',
        'task-plugin',
        'task-plugin-versions',
      ]) {
        void queryClient.invalidateQueries({ queryKey: [key] })
      }
      props.onClose()
    },
    onError: () =>
      toast.error(
        t(
          'Plugin installation failed. Check the source, version conflict and permissions.'
        )
      ),
  })
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !install.isPending) props.onClose()
      }}
    >
      <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle>
            {t('Review and install')} · {props.target.plugin.name}
          </DialogTitle>
          <DialogDescription>
            {t(
              'Review the source before installing. SHA-256 verifies integrity, not publisher identity. New versions stay disabled and inactive.'
            )}
          </DialogDescription>
        </DialogHeader>
        <p className='text-sm break-all'>
          {props.target.source.name} · {version.version}
          <br />
          {props.target.source.index_url}
        </p>
        {download.isPending && <p>{t('Loading...')}</p>}
        {download.isError && (
          <ErrorState
            description={t('Could not download or verify plugin source')}
            onRetry={() => void download.refetch()}
          />
        )}
        {download.data && (
          <Textarea
            aria-label={t('Plugin source')}
            readOnly
            value={download.data.source}
            className='min-h-72 font-mono text-xs'
          />
        )}
        <DialogFooter>
          <Button
            variant='outline'
            disabled={install.isPending}
            onClick={props.onClose}
          >
            {t('Cancel')}
          </Button>
          <Button
            disabled={!download.isSuccess || install.isPending}
            onClick={() => install.mutate()}
          >
            {install.isPending ? t('Installing...') : t('Install')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
