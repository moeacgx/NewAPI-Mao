import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

import { uploadTaskPlugin } from '../api'

const MAX_SOURCE_BYTES = 1 << 20

export function UploadDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialKey?: string
}) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [source, setSource] = useState('')
  const [remark, setRemark] = useState('')
  const mutation = useMutation({
    mutationFn: () => uploadTaskPlugin(source, remark),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['task-plugins'] })
      if (props.initialKey) {
        client.invalidateQueries({
          queryKey: ['task-plugin', props.initialKey],
        })
        client.invalidateQueries({
          queryKey: ['task-plugin-versions', props.initialKey],
        })
      }
      props.onOpenChange(false)
    },
  })
  const tooLarge =
    new TextEncoder().encode(source).byteLength > MAX_SOURCE_BYTES
  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={
        props.initialKey
          ? t('Upload new plugin version')
          : t('Upload task plugin')
      }
      description={t(
        'Review JavaScript source before uploading. Maximum size: 1 MiB.'
      )}
      footer={
        <>
          <Button variant='outline' onClick={() => props.onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button
            disabled={!source.trim() || tooLarge || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {t('Upload')}
          </Button>
        </>
      }
    >
      <div className='space-y-3'>
        <Input
          type='file'
          accept='.js,.mjs,text/javascript'
          onChange={async (event) => {
            const file = event.target.files?.[0]
            if (file) setSource(await file.text())
          }}
        />
        <Textarea
          aria-label={t('Plugin source')}
          rows={14}
          value={source}
          onChange={(event) => setSource(event.target.value)}
        />
        <Input
          aria-label={t('Remark')}
          value={remark}
          placeholder={t('Optional note describing this version')}
          onChange={(event) => setRemark(event.target.value)}
        />
        {tooLarge && (
          <Alert variant='destructive'>
            <AlertDescription>
              {t('Plugin source exceeds the 1 MiB limit.')}
            </AlertDescription>
          </Alert>
        )}
        {mutation.isError && (
          <Alert variant='destructive'>
            <AlertDescription>{mutation.error.message}</AlertDescription>
          </Alert>
        )}
      </div>
    </Dialog>
  )
}
