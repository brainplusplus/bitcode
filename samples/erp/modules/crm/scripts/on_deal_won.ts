import { definePlugin } from '@bitcode/sdk';

export default definePlugin({
  async execute(ctx, params) {
    const lead = params.input;
    
    // Example: send congratulations notification
    console.log(`🎉 Deal Won! ${lead.name} - Revenue: $${lead.expected_revenue}`);
    
    // Example: create a follow-up task
    // await ctx.db.create('activity', {
    //   lead_id: lead.id,
    //   type: 'task',
    //   summary: 'Send welcome package to new client',
    //   due_date: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000)
    // });

    return { success: true, message: 'Deal won notification sent' };
  }
});
