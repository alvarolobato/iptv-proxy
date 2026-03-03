import '@elastic/eui/dist/eui_theme_light.css';
import './icons_hack';

import ReactDOM from 'react-dom/client';
import { EuiProvider } from '@elastic/eui';
import App from './app';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <EuiProvider colorMode="light">
    <App/>
  </EuiProvider>,
);
