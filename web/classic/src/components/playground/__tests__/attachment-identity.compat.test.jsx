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

import React from 'react';
import { act, render } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, expect, test, vi } from 'vitest';
import Playground from '../../../pages/Playground';
import { UserContext } from '../../../context/User';

const capture = vi.hoisted(() => ({
  props: null,
  send: vi.fn(),
  load: vi.fn(),
}));
vi.mock('../ChatArea', () => ({
  default: (props) => {
    capture.props = props;
    return <div>{props.selectedToken?.name || 'Account mode'}</div>;
  },
}));
vi.mock('../OptimizedComponents', () => ({
  OptimizedSettingsPanel: () => null,
  OptimizedDebugPanel: () => null,
  OptimizedMessageContent: () => null,
  OptimizedMessageActions: () => null,
}));
vi.mock('../../../hooks/playground/useDataLoader', () => ({
  useDataLoader: (_user, _inputs, onInputChange, _models, _groups, tokenId) => {
    React.useEffect(() => {
      onInputChange('model', 'model-a');
    }, [onInputChange, tokenId]);
    return { modelsReady: true, modelError: '', loadModels: capture.load };
  },
}));
vi.mock('../../../hooks/playground/useApiRequest', () => ({
  useApiRequest: (_message, _debug, _tab, _sse, _save, tokenId) => ({
    sendRequest: (payload) => capture.send(tokenId, payload),
    onStopGenerator: () => {},
    isRequesting: false,
  }),
}));

afterEach(() => {
  vi.unstubAllGlobals();
  capture.send.mockClear();
});

test.each(['different-token', 'account-round-trip', 'unchanged'])(
  '延迟读取图片后核验身份代次：%s',
  async (change) => {
    const readers = [];
    class DelayedFileReader {
      constructor() {
        readers.push(this);
      }
      readAsDataURL() {}
    }
    vi.stubGlobal('FileReader', DelayedFileReader);
    render(
      <MemoryRouter>
        <UserContext.Provider value={[{ user: { username: 'tester' } }]}>
          <Playground />
        </UserContext.Provider>
      </MemoryRouter>,
    );
    const original = { id: 7, name: 'Original token' };
    act(() => capture.props.onTokenChange(original));
    let pending;
    act(() => {
      pending = capture.props.onMessageSend('Inspect image', [
        {
          fileInstance: new File(['image'], 'sample.png', {
            type: 'image/png',
          }),
        },
      ]);
    });
    expect(readers).toHaveLength(1);
    expect(capture.send).not.toHaveBeenCalled();
    if (change === 'different-token') {
      act(() => capture.props.onTokenChange({ id: 8, name: 'Other token' }));
    } else if (change === 'account-round-trip') {
      act(() => capture.props.onTokenChange(null));
      act(() => capture.props.onTokenChange(original));
    }
    await act(async () => {
      readers[0].result = 'data:image/png;base64,aW1hZ2U=';
      readers[0].onload();
      await pending;
    });
    if (change === 'unchanged') {
      expect(capture.send).toHaveBeenCalledOnce();
      expect(capture.send.mock.calls[0][0]).toBe(7);
      expect(capture.send.mock.calls[0][1].messages[0].content).toContainEqual({
        type: 'image_url',
        image_url: { url: 'data:image/png;base64,aW1hZ2U=' },
      });
    } else {
      expect(capture.send).not.toHaveBeenCalled();
      expect(capture.props.message).toEqual([]);
      expect(capture.props.selectedToken.id).toBe(
        change === 'different-token' ? 8 : 7,
      );
    }
  },
);
