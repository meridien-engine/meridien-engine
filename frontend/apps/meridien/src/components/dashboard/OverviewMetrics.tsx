import { createResource } from 'solid-js';

// Mock JWT for local development (payload contains business_id: 3b825f22-73a8-487f-90d1-794fdf3e81df)
const DEV_JWT = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJidXNpbmVzc19pZCI6IjNiODI1ZjIyLTczYTgtNDg3Zi05MGQxLTc5NGZkZjNlODFkZiJ9.";

type OverviewMetricsData = {
    total_revenue: number;
    orders_processed: number;
    interception_rate: number;
    pending_review: number;
};

const fetchMetrics = async (): Promise<OverviewMetricsData> => {
    const response = await fetch('http://localhost:8080/api/v1/analytics/overview', {
        headers: { 'Authorization': `Bearer ${DEV_JWT}` }
    });
    if (!response.ok) throw new Error('Failed to fetch metrics');
    return response.json();
};

export default function OverviewMetrics() {
    const [metrics] = createResource(fetchMetrics);

    const data = () => metrics() || {
        total_revenue: 0,
        orders_processed: 0,
        interception_rate: 0,
        pending_review: 0,
    };

    return (
        <div class="col-span-12 grid grid-cols-4 gap-6">
            {/* Revenue */}
            <div class="bg-surface-container-low border border-circuit-grey rounded-2xl p-6 relative overflow-hidden group hover:border-logic-teal transition-colors">
                <div class="absolute inset-0 bg-logic-teal/5 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"></div>
                <div class="relative">
                    <p class="text-[11px] font-medium text-on-surface-variant tracking-wide mb-2">TOTAL REVENUE</p>
                    <p class="text-3xl font-bold text-on-surface font-heading tracking-tight">
                        ${data().total_revenue.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                    </p>
                    <div class="flex items-center gap-2 mt-4 pt-4 border-t border-circuit-grey/50">
                        <span class="text-[11px] font-medium text-logic-teal">+8.3% this week</span>
                    </div>
                </div>
            </div>
            
            {/* Orders handled by AI */}
            <div class="bg-surface-container-low border border-circuit-grey rounded-2xl p-6 relative overflow-hidden group hover:border-logic-teal transition-colors">
                <div class="absolute inset-0 bg-logic-teal/5 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"></div>
                <div class="relative">
                    <p class="text-[11px] font-medium text-on-surface-variant tracking-wide mb-2">ORDERS PROCESSED</p>
                    <p class="text-3xl font-bold text-on-surface font-heading tracking-tight">{data().orders_processed}</p>
                    <div class="flex items-center gap-2 mt-4 pt-4 border-t border-circuit-grey/50">
                        <span class="text-[11px] font-medium text-logic-teal">98% Success Rate</span>
                    </div>
                </div>
            </div>

            {/* AI Interception Rate */}
            <div class="bg-surface-container-low border border-circuit-grey rounded-2xl p-6 relative overflow-hidden group hover:border-logic-teal transition-colors">
                <div class="absolute inset-0 bg-logic-teal/5 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"></div>
                <div class="relative">
                    <p class="text-[11px] font-medium text-on-surface-variant tracking-wide mb-2">INTERCEPTION RATE</p>
                    <p class="text-3xl font-bold text-on-surface font-heading tracking-tight">{data().interception_rate.toFixed(1)}%</p>
                    <div class="flex items-center gap-2 mt-4 pt-4 border-t border-circuit-grey/50">
                        <span class="text-[11px] font-medium text-terminal-dim">Of inbound traffic</span>
                    </div>
                </div>
            </div>

            {/* Pending HITL */}
            <div class="bg-surface-container-low border border-warning-amber rounded-2xl p-6 relative overflow-hidden group">
                <div class="relative">
                    <p class="text-[11px] font-medium text-warning-amber tracking-wide mb-2 flex items-center gap-2">
                        PENDING REVIEW
                    </p>
                    <p class="text-3xl font-bold text-warning-amber font-heading tracking-tight">{data().pending_review}</p>
                    <div class="flex items-center gap-2 mt-4 pt-4 border-t border-warning-amber/30">
                        <span class="text-[11px] font-medium text-warning-amber">Requires manual action</span>
                    </div>
                </div>
            </div>
        </div>
    );
}
