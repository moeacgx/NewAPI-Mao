/*
Copyright (C) 2025 QuantumNous

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
import React, { useState } from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createInstance } from 'i18next';
import { I18nextProvider } from 'react-i18next';
import { expect, test } from 'vitest';
import { GroupMultiPicker } from '../modals/EditTokenModal';

const groups = [
  { value: 'auto', label: 'Automatic routing', exclusive: false },
  { value: 'default', label: 'Standard models', exclusive: false },
  { value: 'vip', label: 'Priority models', exclusive: false },
];

function translate(key, options = {}) {
  return String(key)
    .replace('{{count}}', options.count ?? '')
    .replace('{{max}}', options.max ?? '');
}

function LimitPicker({ maxCount, initial }) {
  const [selectedGroups, setSelectedGroups] = useState(initial);
  return (
    <GroupMultiPicker
      groups={groups}
      selectedGroups={selectedGroups}
      maxCount={maxCount}
      onChange={setSelectedGroups}
      t={translate}
    />
  );
}

test('disables another explicit group once the token reaches the configured limit', async () => {
  const i18n = createInstance();
  await i18n.init({
    lng: 'en',
    resources: { en: { translation: {} } },
    interpolation: { escapeValue: false },
  });
  const user = userEvent.setup();
  const view = render(
    <I18nextProvider i18n={i18n}>
      <LimitPicker maxCount={1} initial={['default']} />
    </I18nextProvider>,
  );

  expect(screen.getByText('1 / 1 groups selected')).toBeTruthy();
  await user.click(screen.getByRole('button', { name: /添加分组/ }));
  const blocked = screen.getByRole('button', { name: /Priority models/ });
  expect(blocked.disabled).toBe(true);
  expect(screen.getByText('Standard models')).toBeTruthy();

  view.rerender(
    <I18nextProvider i18n={i18n}>
      <LimitPicker key='over-limit' maxCount={1} initial={['default', 'vip']} />
    </I18nextProvider>,
  );
  expect(screen.getByText('A token can select at most 1 groups')).toBeTruthy();
});
