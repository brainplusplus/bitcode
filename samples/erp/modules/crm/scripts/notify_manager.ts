import { definePlugin } from '@bitcode/sdk';

export default definePlugin({
  async execute(ctx, params) {
    const leadId = params.lead_id;
    console.log(`📧 Notification: Lead ${leadId} has been qualified. Manager review needed.`);
    return { notified: true };
  }
});
