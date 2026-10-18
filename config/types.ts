export enum Environment {
  Local = 'local',
  Development = 'development',
  Production = 'production',
  Staging = 'staging',
}

export interface Config {
	REFERENCE_DETAILS_AVAILABLE?: boolean;
  API_HOST: string;
  ENVIRONMENT: string;
  GA_ID: string;
  ROLLBAR_TOKEN: string;
  VERSION: string;
}
