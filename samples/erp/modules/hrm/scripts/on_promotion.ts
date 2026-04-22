import { definePlugin } from '@bitcode/sdk';

export default definePlugin({
  async execute(ctx, params) {
    console.log(`🎉 Employee ${params.employee_id} has been promoted!`);
    console.log('Sending congratulations email and updating org chart...');
    return { notified: true };
  }
});
