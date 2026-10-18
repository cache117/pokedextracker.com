import { Development } from './development';
import { Local } from './local';
import { Production } from './production';
import { Staging } from './staging';
import { Environment } from './types';

import type { Config as ConfigType } from './types';

const ENVIRONMENTS: Record<string, Environment> = {
  'pokedextracker.com': Environment.Production,
  'staging.pokedextracker.com': Environment.Staging,
};

const environment = ENVIRONMENTS[window.location.hostname] || Environment.Local;

const Configs: Record<Environment, ConfigType> = {
  [Environment.Local]: Local,
  [Environment.Development]: Development,
  [Environment.Production]: Production,
  [Environment.Staging]: Staging,
};

export const Config = Configs[environment] as ConfigType;
