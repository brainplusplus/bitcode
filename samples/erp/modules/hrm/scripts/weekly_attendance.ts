import { definePlugin } from '@bitcode/sdk';

export default definePlugin({
  async execute(ctx, params) {
    console.log('📊 Generating weekly attendance report...');
    return { report: 'generated' };
  }
});
