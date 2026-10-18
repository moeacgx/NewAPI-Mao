type SiteMetadataStatus = {
  system_name?: unknown
  system_description?: unknown
}

function updateSingleMeta(
  selector: string,
  attribute: string,
  value: string,
  content: string
) {
  const existing = [...document.head.querySelectorAll(selector)]
  const meta = existing.shift() ?? document.createElement('meta')
  existing.forEach((item) => item.remove())
  meta.setAttribute(attribute, value)
  meta.setAttribute('content', content)
  if (!meta.isConnected) document.head.appendChild(meta)
}

function removeMeta(selector: string) {
  document.head.querySelectorAll(selector).forEach((meta) => meta.remove())
}

export function applySiteMetadataFromStatus(status: SiteMetadataStatus) {
  if (typeof document === 'undefined') return

  if (typeof status.system_name === 'string' && status.system_name !== '') {
    document.title = status.system_name
    const titles = [...document.head.querySelectorAll('title')]
    const title = titles.shift() ?? document.createElement('title')
    titles.forEach((item) => item.remove())
    title.textContent = status.system_name
    if (!title.isConnected) document.head.appendChild(title)

    updateSingleMeta('meta[name="title"]', 'name', 'title', status.system_name)
    updateSingleMeta(
      'meta[property="og:title"]',
      'property',
      'og:title',
      status.system_name
    )
    updateSingleMeta(
      'meta[property="og:site_name"]',
      'property',
      'og:site_name',
      status.system_name
    )
  }

  if (Object.hasOwn(status, 'system_description')) {
    if (
      typeof status.system_description === 'string' &&
      status.system_description !== ''
    ) {
      updateSingleMeta(
        'meta[name="description"]',
        'name',
        'description',
        status.system_description
      )
      updateSingleMeta(
        'meta[property="og:description"]',
        'property',
        'og:description',
        status.system_description
      )
    } else {
      removeMeta('meta[name="description"], meta[property="og:description"]')
    }
  }
}
