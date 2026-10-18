import './styles';

import Modal from 'react-modal';
import { QueryClientProvider } from '@tanstack/react-query';
import { render } from 'react-dom';

import { App } from './components/pages/App';
import { LocalStorageContextProvider } from './hooks/contexts/use-local-storage-context';
import { SessionProvider } from './hooks/contexts/use-session';
import { queryClient } from './utils/query-client';

Modal.setAppElement('#root');

function run () {
  render(
        <QueryClientProvider client={queryClient}>
          <SessionProvider>
            <LocalStorageContextProvider>
              <App />
            </LocalStorageContextProvider>
          </SessionProvider>
        </QueryClientProvider>,
    document.getElementById('root')
  );
}

if (document.getElementById('root')) {
  run();
} else {
  window.addEventListener('DOMContentLoaded', run);
}
