/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { MESSAGE_ROLES, MESSAGE_STATUS } from '../../../constants'
import { createRegeneratedMessages } from '../../../lib/message/conversation-message-utils'
import { formatMessageForAPI } from '../../../lib/message/message-utils'
import { messagesSchema } from '../../../lib/storage/storage-schema'
import type { Message } from '../../../types'
import { MessageMetadata } from '../message-metadata'

const assistantMessage: Message = {
  key: 'assistant-1',
  from: MESSAGE_ROLES.ASSISTANT,
  versions: [{ id: 'version-1', content: 'Done' }],
  status: MESSAGE_STATUS.COMPLETE,
  gatewaySkills: {
    version: 1,
    total: 2,
    skills: [
      {
        templateId: 10,
        skillId: 'review',
        version: 1,
        title: 'Code review',
      },
      {
        templateId: 11,
        skillId: 'docs',
        version: 2,
        title: 'Documentation',
      },
    ],
    truncated: false,
  },
}

describe('gateway skills message metadata', () => {
  it('renders a compact gateway skills badge below a completed assistant reply', () => {
    render(<MessageMetadata alignment='left' message={assistantMessage} />)

    expect(screen.getByText('Gateway cloud Skills')).toBeInTheDocument()
    expect(screen.getByText('Code review, Documentation')).toBeInTheDocument()
    expect(
      screen.getByLabelText('Gateway cloud Skills: Code review, Documentation')
    ).toBeInTheDocument()
  })

  it('does not render stale metadata while an assistant reply is loading or failed', () => {
    const { rerender } = render(
      <MessageMetadata
        alignment='left'
        message={{ ...assistantMessage, status: MESSAGE_STATUS.LOADING }}
      />
    )
    expect(screen.queryByText('Gateway cloud Skills')).not.toBeInTheDocument()

    rerender(
      <MessageMetadata
        alignment='left'
        message={{ ...assistantMessage, status: MESSAGE_STATUS.ERROR }}
      />
    )
    expect(screen.queryByText('Gateway cloud Skills')).not.toBeInTheDocument()
  })

  it('persists usage metadata without adding it to model messages or regenerated replies', () => {
    expect(messagesSchema.parse([assistantMessage])).toEqual([assistantMessage])
    expect(formatMessageForAPI(assistantMessage)).toEqual({
      role: 'assistant',
      content: 'Done',
    })

    const regenerated = createRegeneratedMessages(
      [
        {
          key: 'user-1',
          from: MESSAGE_ROLES.USER,
          versions: [{ id: 'user-version', content: 'Review this' }],
        },
        assistantMessage,
      ],
      assistantMessage.key
    )

    expect(regenerated?.at(-1)).not.toHaveProperty('gatewaySkills')
  })
})
