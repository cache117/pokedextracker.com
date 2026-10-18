import type { Config } from './types';

export const Local: Config = {
  REFERENCE_DETAILS_AVAILABLE: true,
  API_HOST: '/api',
  ENVIRONMENT: 'development',
  GA_ID: '',
  ROLLBAR_TOKEN: '',
  VERSION: process.env.VERSION || 'unknown',
};
