import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import { uploadTaskPlugin } from '../api'

type UploadPluginDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

function uploadErrorMessage(error: unknown, fallback: string) {
  const status = (error as { response?: { status?: number } })?.response?.status
  if (status === 409) {
    return 'A plugin with this key and version already exists with different source.'
  }
  if (status === 413) {
    return 'Plugin source must be 1 MiB or smaller.'
  }
  return fallback
}

export function UploadPluginDialog(props: UploadPluginDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [source, setSource] = useState('')
  const upload = useMutation({
    mutationFn: () => uploadTaskPlugin(source),
    onSuccess: () => {
      toast.success(t('Plugin uploaded'))
      setSource('')
      props.onOpenChange(false)
      void queryClient.invalidateQueries({ queryKey: ['task-plugins'] })
      void queryClient.invalidateQueries({ queryKey: ['task-plugin-options'] })
    },
    onError: (error) => {
      toast.error(uploadErrorMessage(error, t('Plugin upload failed')))
    },
  })

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>{t('Upload custom task plugin')}</DialogTitle>
          <DialogDescription>
            {t(
              'Only Root users can upload. The source is compiled and stored disabled and inactive.'
            )}
          </DialogDescription>
        </DialogHeader>
        <div className='space-y-2'>
          <Label htmlFor='task-plugin-source'>{t('Plugin source')}</Label>
          <Textarea
            id='task-plugin-source'
            value={source}
            onChange={(event) => setSource(event.target.value)}
            placeholder={t('Paste JavaScript plugin source')}
            className='min-h-72 font-mono text-xs'
            maxLength={1024 * 1024}
            disabled={upload.isPending}
          />
          <p className='text-muted-foreground text-xs'>
            {t(
              'Maximum source size: 1 MiB. The plugin key and version come from its metadata.'
            )}
          </p>
        </div>
        <DialogFooter>
          <Button
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={upload.isPending}
          >
            {t('Cancel')}
          </Button>
          <Button
            onClick={() => upload.mutate()}
            disabled={!source.trim() || upload.isPending}
          >
            {upload.isPending ? t('Uploading') : t('Upload')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
