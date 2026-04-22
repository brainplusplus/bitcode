import { Config } from '@stencil/core';

export const config: Config = {
  namespace: 'bc-components',
  globalScript: 'src/i18n/index.ts',
  outputTargets: [
    {
      type: 'dist',
      esmLoaderPath: '../loader',
    },
    {
      type: 'dist-custom-elements',
    },
    {
      type: 'www',
      serviceWorker: null,
    },
  ],
  globalStyle: 'src/global/global.css',
  testing: {
    browserHeadless: 'new',
  },
};
