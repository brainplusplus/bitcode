import { definePlugin } from '@bitcode/sdk';

export default definePlugin({
  async execute(ctx, params) {
    console.log('📊 Weekly Pipeline Report generated');
    // Example: query leads by status, calculate totals, send email
    return { report: 'generated' };
  }
});
