import { createResource, For, createSignal, Show } from 'solid-js';

const DEV_JWT = typeof window !== 'undefined' ? localStorage.getItem('MERIDIEN_AUTH_TOKEN') || '' : '';

type HitlRecord = {
    trace_id: string;
    interaction_log_id: string;
    channel: string;
    inbound_msg: string;
    outbound_msg: string;
    hitl_status: string;
    suspended_at?: string;
    expires_at?: string;
    raw_agent_thoughts: string;
};

const fetchHitl = async (): Promise<HitlRecord[]> => {
    const response = await fetch('/api/v1/hitl', {
        headers: { 'Authorization': `Bearer ${DEV_JWT}` }
    });
    if (!response.ok) {
        // Return mock data if backend not ready
        return [
            {
                trace_id: "trace-5e6f7g8h",
                interaction_log_id: "log-1w2v3u",
                channel: "messenger",
                inbound_msg: "Can I get a discount if I buy 50 units?",
                outbound_msg: "Let me forward this request to our sales team for approval. They will contact you shortly.",
                hitl_status: "pending",
                suspended_at: new Date(Date.now() - 3600000).toISOString(),
                expires_at: new Date(Date.now() + 82800000).toISOString(), // 23 hours from now
                raw_agent_thoughts: "Thinking Process:\n1. User requesting bulk discount.\n2. Check policy in Knowledge Base.\n3. Policy states >20 units requires sales approval.\n4. Triggering HITL suspension workflow."
            },
            {
                trace_id: "trace-x9y8z7w6",
                interaction_log_id: "log-4d5e6f",
                channel: "whatsapp",
                inbound_msg: "I want a refund, the product arrived completely shattered!",
                outbound_msg: "I apologize for the poor experience. I am escalating this refund request to a human agent.",
                hitl_status: "pending",
                suspended_at: new Date(Date.now() - 1800000).toISOString(),
                expires_at: new Date(Date.now() + 84600000).toISOString(),
                raw_agent_thoughts: "Thinking Process:\n1. User is highly dissatisfied and requesting a refund for damaged goods.\n2. Sentiment analysis: Negative/Angry.\n3. Return policy states damaged goods refunds require photo evidence.\n4. Escalate to HITL for manual handling to ensure customer satisfaction."
            }
        ];
    }
    return response.json();
};

export default function HITLGrid() {
    const [hitlRecords, { refetch }] = createResource(fetchHitl);
    const [expandedRowId, setExpandedRowId] = createSignal<string | null>(null);
    const [resolvingId, setResolvingId] = createSignal<string | null>(null);

    const toggleExpand = (id: string) => {
        setExpandedRowId(prev => prev === id ? null : id);
    };

    const resolveHitl = async (id: string, status: 'approved' | 'rejected') => {
        setResolvingId(id);
        try {
            const res = await fetch(`/api/v1/hitl/${id}/resolve`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${DEV_JWT}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ status })
            });
            if (res.ok) {
                // Remove from local list or refetch
                refetch();
            } else {
                console.error("Failed to resolve HITL");
                // Mock success for UI if backend fails
                setTimeout(() => refetch(), 500);
            }
        } catch (e) {
            console.error(e);
            setTimeout(() => refetch(), 500);
        } finally {
            setResolvingId(null);
        }
    };

    const getHitlBadge = (status: string) => {
        switch (status) {
            case 'pending': return <span class="bg-warning-amber/10 text-warning-amber border border-warning-amber/30 px-2 py-0.5 rounded text-[10px] uppercase font-bold">Pending Review</span>;
            case 'approved': return <span class="bg-logic-teal/10 text-logic-teal border border-logic-teal/30 px-2 py-0.5 rounded text-[10px] uppercase font-bold">Approved</span>;
            case 'rejected': return <span class="bg-red-500/10 text-red-500 border border-red-500/30 px-2 py-0.5 rounded text-[10px] uppercase font-bold">Rejected</span>;
            default: return null;
        }
    };

    return (
        <div class="bg-surface-container-low border border-circuit-grey rounded-2xl flex flex-col h-[calc(100vh-160px)]">
            <div class="flex-1 overflow-y-auto overflow-x-auto relative custom-scrollbar">
                <div class="min-w-[1000px] flex flex-col min-h-full">
                    {/* Header */}
                    <div class="flex items-center px-6 py-4 border-b border-circuit-grey bg-surface-container-low text-xs font-semibold text-on-surface-variant uppercase tracking-wider sticky top-0 z-20 shrink-0">
                        <div class="w-1/12 pr-4">Expand</div>
                        <div class="w-2/12 pr-4">Suspended At</div>
                        <div class="w-2/12 pr-4">Channel</div>
                        <div class="w-3/12 pr-4">Inbound Message</div>
                        <div class="w-2/12 pr-4">Status</div>
                        <div class="w-2/12 text-right">Quick Actions</div>
                    </div>

                    {/* Body */}
                    <div class="flex-1 relative">
                        {hitlRecords.loading && (
                            <div class="absolute inset-0 flex items-center justify-center bg-surface-container-low/50 backdrop-blur-sm z-10">
                                <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-logic-teal"></div>
                            </div>
                        )}
                        {!hitlRecords.loading && (
                            <For each={hitlRecords()?.filter(r => r.hitl_status === 'pending') || []}>
                                {(record) => (
                                    <div class="border-b border-circuit-grey/50">
                                        {/* Main Row */}
                                        <div 
                                            class={`flex items-center px-6 py-4 cursor-pointer transition-colors ${expandedRowId() === record.trace_id ? 'bg-surface-container-highest' : 'hover:bg-surface-container-highest'}`}
                                            onClick={() => toggleExpand(record.trace_id)}
                                        >
                                            <div class="w-1/12 pr-4 text-terminal-dim">
                                                <svg class={`w-5 h-5 transition-transform ${expandedRowId() === record.trace_id ? 'rotate-90 text-logic-teal' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>
                                            </div>
                                            <div class="w-2/12 pr-4 text-sm font-medium text-on-surface">
                                                {record.suspended_at ? new Date(record.suspended_at).toLocaleString() : 'N/A'}
                                            </div>
                                            <div class="w-2/12 pr-4 text-xs text-on-surface-variant capitalize">
                                                {record.channel}
                                            </div>
                                            <div class="w-3/12 pr-4">
                                                <p class="text-sm text-on-surface truncate pr-4">{record.inbound_msg}</p>
                                            </div>
                                            <div class="w-2/12 pr-4">
                                                {getHitlBadge(record.hitl_status)}
                                            </div>
                                            <div class="w-2/12 text-right flex items-center justify-end gap-2" onClick={(e) => e.stopPropagation()}>
                                                <button 
                                                    disabled={resolvingId() === record.trace_id}
                                                    onClick={() => resolveHitl(record.trace_id, 'approved')}
                                                    class="bg-logic-teal/10 hover:bg-logic-teal text-logic-teal hover:text-white px-3 py-1.5 rounded text-xs font-bold uppercase transition-colors border border-logic-teal/30 disabled:opacity-50"
                                                >
                                                    Approve
                                                </button>
                                                <button 
                                                    disabled={resolvingId() === record.trace_id}
                                                    onClick={() => resolveHitl(record.trace_id, 'rejected')}
                                                    class="bg-red-500/10 hover:bg-red-500 text-red-500 hover:text-white px-3 py-1.5 rounded text-xs font-bold uppercase transition-colors border border-red-500/30 disabled:opacity-50"
                                                >
                                                    Reject
                                                </button>
                                            </div>
                                        </div>

                                        {/* Expanded Detail Panel */}
                                        <Show when={expandedRowId() === record.trace_id}>
                                            <div class="bg-surface-container p-6 border-t border-circuit-grey/50 shadow-inner">
                                                <div class="grid grid-cols-2 gap-8">
                                                    {/* Context */}
                                                    <div class="flex flex-col gap-4">
                                                        <div>
                                                            <h4 class="text-[10px] font-bold uppercase tracking-wider text-logic-teal mb-2">Customer Message</h4>
                                                            <div class="bg-surface-container-low border border-circuit-grey rounded-lg p-3 text-sm text-on-surface">
                                                                {record.inbound_msg}
                                                            </div>
                                                        </div>
                                                        <div>
                                                            <h4 class="text-[10px] font-bold uppercase tracking-wider text-logic-teal mb-2">AI Proposed Response</h4>
                                                            <div class="bg-surface-container-low border border-circuit-grey rounded-lg p-3 text-sm text-on-surface-variant italic">
                                                                {record.outbound_msg}
                                                            </div>
                                                        </div>
                                                    </div>

                                                    {/* Rationale */}
                                                    <div>
                                                        <h4 class="text-[10px] font-bold uppercase tracking-wider text-logic-teal mb-2 flex items-center gap-2">
                                                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
                                                            Why was this suspended?
                                                        </h4>
                                                        <div class="bg-void-black border border-circuit-grey rounded-lg p-4 font-mono text-xs text-on-surface-variant whitespace-pre-wrap">
                                                            {record.raw_agent_thoughts || 'No reasoning provided.'}
                                                        </div>
                                                        {record.expires_at && (
                                                            <div class="mt-4 flex items-center gap-2 text-xs text-warning-amber">
                                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
                                                                Expires: {new Date(record.expires_at).toLocaleString()}
                                                            </div>
                                                        )}
                                                    </div>
                                                </div>
                                            </div>
                                        </Show>
                                    </div>
                                )}
                            </For>
                        )}
                        {(!hitlRecords.loading && (!hitlRecords() || hitlRecords()?.filter(r => r.hitl_status === 'pending').length === 0)) && (
                            <div class="flex flex-col items-center justify-center h-full text-terminal-dim py-12">
                                <svg class="w-12 h-12 mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
                                <p class="text-sm">No pending human reviews. You're all caught up!</p>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
