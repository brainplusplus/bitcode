import { definePlugin } from '@bitcode/sdk';

export default definePlugin({
  async execute(ctx, params) {
    console.log('⏰ Checking for stale leads (no activity in 7 days)...');
    // Example: query leads with no recent activity, send reminders
    return { checked: true };
  }
});
