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
import { Layers3 } from 'lucide-react'
import { lazy, Suspense, useState } from 'react'

const GroupTagIcon = lazy(() => import('./group-tag-icon'))

export function GroupTagLogos(props: { icons: string[] }) {
  return (
    <span className='flex flex-wrap items-center gap-1' aria-hidden='true'>
      {props.icons.length === 0 ? (
        <Layers3 className='size-4' />
      ) : (
        props.icons.map((icon) => <GroupTagLogo key={icon} icon={icon} />)
      )}
    </span>
  )
}

function GroupTagLogo(props: { icon: string }) {
  const [failed, setFailed] = useState(false)
  if (failed) return <Layers3 className='size-4' />
  if (/^https?:\/\//i.test(props.icon)) {
    return (
      <img
        src={props.icon}
        alt=''
        loading='lazy'
        referrerPolicy='no-referrer'
        className='size-5 object-contain'
        onError={() => setFailed(true)}
      />
    )
  }
  return (
    <Suspense fallback={<Layers3 className='size-4' />}>
      <GroupTagIcon icon={props.icon} />
    </Suspense>
  )
}
