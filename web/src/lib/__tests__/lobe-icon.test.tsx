import { fireEvent, render, screen } from '@testing-library/react'
import React from 'react'
import { describe, expect, it, vi } from 'vitest'

import { getLobeIcon } from '../lobe-icon'

vi.mock('@lobehub/icons', () => {
  const credentialedUrlBase = ['https://user:', 'secret', '@cdn'].join('')
  const icons: Record<string, unknown> = {
    OpenAI: { Avatar: () => null },
    'javascript:alert(1)': undefined,
    'https://user@cdn': undefined,
    [credentialedUrlBase]: undefined,
    'https://cdn': undefined,
    'https://cdn@cdn': undefined,
    'https://:': undefined,
    'https://:@cdn': undefined,
  }
  return new Proxy(icons, {
    get: (target, property: string) =>
      property.includes('@') ? undefined : target[property],
  })
})

describe('getLobeIcon custom image URLs', () => {
  it.each([
    'https://cdn.example.com/logo.svg',
    'http://cdn.example.com/logo.png',
  ])('renders an HTTP or HTTPS image URL at a fixed size: %s', (url) => {
    const { container } = render(<>{getLobeIcon(url, 28)}</>)
    const image = container.querySelector('img')

    expect(image).toHaveAttribute('src', url)
    expect(image).toHaveAttribute('width', '28')
    expect(image).toHaveAttribute('height', '28')
    expect(image).toHaveAttribute('loading', 'lazy')
    expect(image).toHaveAttribute('referrerpolicy', 'no-referrer')
    expect(image).toHaveStyle({
      width: '28px',
      height: '28px',
      objectFit: 'contain',
      flexShrink: '0',
    })
  })

  it('falls back to a fixed-size question mark when an image fails', () => {
    const { container } = render(
      <>{getLobeIcon('https://cdn.example.com/logo.svg', 24)}</>
    )

    fireEvent.error(container.querySelector('img') as HTMLImageElement)

    expect(screen.getByText('?')).toHaveStyle({
      width: '24px',
      height: '24px',
      flexShrink: '0',
    })
  })

  it('resets the image failure state when the URL changes', () => {
    const { container, rerender } = render(
      <>{getLobeIcon('https://cdn.example.com/old.svg', 20)}</>
    )
    fireEvent.error(container.querySelector('img') as HTMLImageElement)
    expect(screen.getByText('?')).toBeInTheDocument()

    rerender(<>{getLobeIcon('https://cdn.example.com/new.svg', 20)}</>)

    expect(container.querySelector('img')).toHaveAttribute(
      'src',
      'https://cdn.example.com/new.svg'
    )
  })

  it('keeps empty and non-HTTP values on the text fallback path', () => {
    const { container, rerender } = render(<>{getLobeIcon('', 18)}</>)
    expect(screen.getByText('?')).toBeInTheDocument()

    rerender(<>{getLobeIcon('javascript:alert(1)', 18)}</>)

    expect(container.querySelector('img')).toBeNull()
    expect(screen.getByText('J')).toBeInTheDocument()
  })

  it.each([
    'https://user@cdn.example.com/logo.svg',
    ['https://user', ':', 'secret', '@cdn.example.com/logo.svg'].join(''),
    'https://cdn.example.com@cdn.example.com/logo.svg',
    'https://cdn@cdn.example.com/logo.svg',
    'https://:@cdn.example.com/logo.svg',
  ])('does not load a URL containing credentials: %s', (url) => {
    const { container } = render(<>{getLobeIcon(url, 18)}</>)

    expect(container.querySelector('img')).toBeNull()
    expect(screen.getByText('H')).toBeInTheDocument()
  })

  it('preserves LobeHub chained properties and custom icon components', () => {
    const chainedIcon = getLobeIcon("OpenAI.Avatar.type={'platform'}", 24)
    expect(React.isValidElement(chainedIcon)).toBe(true)
    expect(chainedIcon).toMatchObject({
      props: { type: 'platform', size: 24 },
    })

    const customIcon = getLobeIcon('Sub2API', 24)
    expect(React.isValidElement(customIcon)).toBe(true)
    expect(customIcon).toMatchObject({ props: { size: 24 } })
  })
})
