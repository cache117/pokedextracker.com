import RealGA from 'react-ga';

import { Config } from '../../config';

const ReactGA = Config.ENVIRONMENT === 'development'
  ? {
    event: (...args: Parameters<typeof RealGA.event>) => {
      void args;
    },
    set: (...args: Parameters<typeof RealGA.set>) => {
      void args;
    },
    pageview: (...args: Parameters<typeof RealGA.pageview>) => {
      void args;
    },
  }
  : RealGA;

if (Config.ENVIRONMENT !== 'development') {
  RealGA.initialize(Config.GA_ID, { titleCase: false });
}

export { ReactGA };

export function logPageView () {
  ReactGA.set({ page: window.location.pathname });
  ReactGA.pageview(window.location.pathname);
}
