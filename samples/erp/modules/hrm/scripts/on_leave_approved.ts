import { definePlugin } from '@bitcode/sdk';

export default definePlugin({
  async execute(ctx, params) {
    console.log(`✅ Leave approved. Deducting ${params.input?.days || 0} days from leave balance.`);
    // In production: update employee.leave_balance -= days
    return { balance_updated: true };
  }
});
