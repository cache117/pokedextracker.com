import RealGA from 'react-ga';

import { Config } from '../../config';

const ReactGA = Config.ENVIRONMENT === 'development'
  ? {
    event: (..._args: Parameters<typeof RealGA.event>) => {},
    set: (..._args: Parameters<typeof RealGA.set>) => {},
    pageview: (..._args: Parameters<typeof RealGA.pageview>) => {},
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
