/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { useAuthStore } from '@/stores/auth-store'

import { createChannel, updateChannel } from '../api'
import { ERROR_MESSAGES, SUCCESS_MESSAGES } from '../constants'
import {
  transformFormDataToCreatePayload,
  transformFormDataToUpdatePayload,
  type ChannelFormValues,
} from '../lib'
import {
  buildMultiKeyConversionFields,
  canConvertChannelToMultiKey,
  isIonetManagedChannel,
} from '../lib/multi-key-conversion'
import type { Channel } from '../types'

type UseChannelMutateFormParams = {
  currentRow?: Channel | null
  isEditing: boolean
  isMultiKeyChannel: boolean
  onSuccess: () => void
}

const SENSITIVE_UPDATE_FIELDS = [
  'type',
  'key',
  'base_url',
  'openai_organization',
  'param_override',
  'header_override',
  'setting',
  'settings',
  'other',
] satisfies (keyof Channel)[]

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function getErrorMessage(error: unknown): string | undefined {
  if (error instanceof Error && typeof error.message === 'string') {
    return error.message
  }

  if (!isRecord(error)) return undefined

  const response = error.response
  if (isRecord(response)) {
    const data = response.data
    if (isRecord(data)) {
      const message = data.message
      if (typeof message === 'string') return message
    }
  }

  const message = error.message
  if (typeof message === 'string') return message
  return undefined
}

export function useChannelMutateForm(props: UseChannelMutateFormParams) {
  const { t } = useTranslation()
  const currentUser = useAuthStore((s) => s.auth.user)
  const canWrite = hasPermission(
    currentUser,
    ADMIN_PERMISSION_RESOURCES.CHANNEL,
    ADMIN_PERMISSION_ACTIONS.WRITE
  )
  const canEditSensitive = hasPermission(
    currentUser,
    ADMIN_PERMISSION_RESOURCES.CHANNEL,
    ADMIN_PERMISSION_ACTIONS.SENSITIVE_WRITE
  )

  return useMutation({
    mutationFn: async (data: ChannelFormValues): Promise<string> => {
      if (props.isEditing ? !canWrite : !canEditSensitive) {
        throw new Error(t("You don't have necessary permission"))
      }
      if (props.isEditing && props.currentRow) {
        if (
          data.convert_to_multi_key &&
          (!canEditSensitive ||
            !canConvertChannelToMultiKey({
              isEditing: props.isEditing,
              isMultiKeyChannel: props.isMultiKeyChannel,
              originalType: props.currentRow.type,
              type: data.type,
              isIonetLocked: isIonetManagedChannel(props.currentRow.other_info),
            }))
        ) {
          throw new Error(t("You don't have necessary permission"))
        }
        const payload = transformFormDataToUpdatePayload(
          data,
          props.currentRow.id
        )
        if (!data.key?.trim()) {
          delete payload.key
        }
        if (!canEditSensitive) {
          for (const field of SENSITIVE_UPDATE_FIELDS) {
            delete payload[field]
          }
        }
        const response = await updateChannel(props.currentRow.id, {
          ...payload,
          ...buildMultiKeyConversionFields({
            isMultiKeyChannel: props.isMultiKeyChannel,
            convertToMultiKey: data.convert_to_multi_key,
            canWriteSensitive: canEditSensitive,
            hasNewKey: Boolean(data.key?.trim()),
            keyMode: data.key_mode,
            multiKeyMode: data.multi_key_type,
          }),
        })
        if (!response.success) {
          throw new Error(response.message || t(ERROR_MESSAGES.UPDATE_FAILED))
        }
        return SUCCESS_MESSAGES.UPDATED
      }

      const payload = transformFormDataToCreatePayload(data)
      const response = await createChannel(payload)
      if (!response.success) {
        throw new Error(response.message || t(ERROR_MESSAGES.CREATE_FAILED))
      }
      return SUCCESS_MESSAGES.CREATED
    },
    onSuccess: (messageKey) => {
      toast.success(t(messageKey))
      props.onSuccess()
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error) || t(ERROR_MESSAGES.CREATE_FAILED))
    },
  })
}
