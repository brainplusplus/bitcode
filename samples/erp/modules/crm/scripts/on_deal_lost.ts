import { definePlugin } from '@bitcode/sdk';

export default definePlugin({
  async execute(ctx, params) {
    const lead = params.input;
    console.log(`❌ Deal Lost: ${lead.name} - Reason: ${lead.lost_reason}`);
    return { success: true };
  }
});
