import { describe, expect, it } from 'vitest'

import { applySiteMetadataFromStatus } from './site-metadata'

describe('site metadata', () => {
  it('applies the newest server branding and description to the document', () => {
    document.head.innerHTML = `
      <title>Server site name</title>
      <meta name="title" content="Server site name">
      <meta name="description" content="Old static description">
      <meta property="og:title" content="Server site name">
      <meta property="og:site_name" content="Server site name">
      <meta property="og:description" content="Old static description">
    `

    applySiteMetadataFromStatus({
      system_name: 'Current site name',
      system_description: '当前站点描述',
    })

    expect(document.title).toBe('Current site name')
    expect(
      document.head.querySelector('meta[name="title"]')?.getAttribute('content')
    ).toBe('Current site name')
    expect(
      document.head
        .querySelector('meta[property="og:title"]')
        ?.getAttribute('content')
    ).toBe('Current site name')
    expect(
      document.head
        .querySelector('meta[property="og:site_name"]')
        ?.getAttribute('content')
    ).toBe('Current site name')
    expect(
      document.head
        .querySelector('meta[name="description"]')
        ?.getAttribute('content')
    ).toBe('当前站点描述')
    expect(
      document.head
        .querySelector('meta[property="og:description"]')
        ?.getAttribute('content')
    ).toBe('当前站点描述')
  })

  it('removes stale description metadata without replacing the main title', () => {
    document.head.innerHTML = `
      <title>Current site name</title>
      <meta name="description" content="Old static description">
      <meta property="og:description" content="Old static description">
    `

    applySiteMetadataFromStatus({
      system_name: 'Current site name',
      system_description: '',
    })

    expect(document.title).toBe('Current site name')
    expect(document.head.querySelector('meta[name="description"]')).toBeNull()
    expect(
      document.head.querySelector('meta[property="og:description"]')
    ).toBeNull()
  })
})
