import { definePlugin } from '@bitcode/sdk';

export default definePlugin({
  async execute(ctx, params) {
    console.log(`📋 Leave request submitted by employee ${params.employee_id} for ${params.days} days`);
    console.log('Notifying direct manager for approval...');
    return { notified: true };
  }
});
