import { createResource, For, createSignal, Show } from 'solid-js';

const DEV_JWT = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJidXNpbmVzc19pZCI6IjNiODI1ZjIyLTczYTgtNDg3Zi05MGQxLTc5NGZkZjNlODFkZiJ9.";

type TraceRecord = {
    trace_id: string;
    interaction_log_id: string;
    channel: string;
    inbound_msg: string;
    outbound_msg: string;
    tokens_used: number;
    latency_ms: number;
    trace_created_at: string;
    hitl_status?: string;
    raw_agent_thoughts: string;
    tools_called: any[];
    retrieved_contexts: any[];
};

const fetchTraces = async (): Promise<TraceRecord[]> => {
    const response = await fetch('/api/v1/traces', {
        headers: { 'Authorization': `Bearer ${DEV_JWT}` }
    });
    if (!response.ok) {
        // If the backend isn't ready yet, return mock data for UI testing
        return [
            {
                trace_id: "trace-1a2b3c4d",
                interaction_log_id: "log-9x8y7z",
                channel: "whatsapp",
                inbound_msg: "Where is my order? I ordered 2 days ago.",
                outbound_msg: "Your order #4912 is currently out for delivery and should arrive by 8 PM today.",
                tokens_used: 432,
                latency_ms: 1250,
                trace_created_at: new Date().toISOString(),
                hitl_status: "none",
                raw_agent_thoughts: "Thinking Process:\n1. User is asking for order status.\n2. Need to extract order info using customer ID.\n3. Found order #4912 in ERP.\n4. Status is 'Out for Delivery'.\n5. Formulate polite response.",
                tools_called: [{ name: "lookup_order_by_customer", args: { customer_id: "cus-123" }, result: "Order 4912, Status: Out for delivery" }],
                retrieved_contexts: []
            },
            {
                trace_id: "trace-5e6f7g8h",
                interaction_log_id: "log-1w2v3u",
                channel: "messenger",
                inbound_msg: "Can I get a discount if I buy 50 units?",
                outbound_msg: "Let me forward this request to our sales team for approval. They will contact you shortly.",
                tokens_used: 512,
                latency_ms: 2100,
                trace_created_at: new Date(Date.now() - 3600000).toISOString(),
                hitl_status: "pending",
                raw_agent_thoughts: "Thinking Process:\n1. User requesting bulk discount.\n2. Check policy in Knowledge Base.\n3. Policy states >20 units requires sales approval.\n4. Triggering HITL suspension workflow.",
                tools_called: [{ name: "search_knowledge_base", args: { query: "bulk discount policy" }, result: "Discounts for >20 units require human approval." }],
                retrieved_contexts: [{ content: "All bulk orders exceeding 20 units must be routed to the sales team for margin review.", score: 0.91 }]
            }
        ];
    }
    return response.json();
};

export default function TraceGrid() {
    const [traces, { refetch }] = createResource(fetchTraces);
    const [expandedTraceId, setExpandedTraceId] = createSignal<string | null>(null);

    const toggleExpand = (id: string) => {
        setExpandedTraceId(prev => prev === id ? null : id);
    };

    const getHitlBadge = (status: string) => {
        switch (status) {
            case 'pending': return <span class="bg-warning-amber/10 text-warning-amber border border-warning-amber/30 px-2 py-0.5 rounded text-[10px] uppercase font-bold">Pending HITL</span>;
            case 'approved': return <span class="bg-logic-teal/10 text-logic-teal border border-logic-teal/30 px-2 py-0.5 rounded text-[10px] uppercase font-bold">Approved</span>;
            case 'rejected': return <span class="bg-red-500/10 text-red-500 border border-red-500/30 px-2 py-0.5 rounded text-[10px] uppercase font-bold">Rejected</span>;
            default: return null;
        }
    };

    return (
        <div class="bg-surface-container-low border border-circuit-grey rounded-2xl flex flex-col h-[calc(100vh-160px)]">
            {/* Scrollable Area */}
            <div class="flex-1 overflow-y-auto overflow-x-auto relative custom-scrollbar">
                <div class="min-w-[1000px] flex flex-col min-h-full">
                    
                    {/* Header */}
                    <div class="flex items-center px-6 py-4 border-b border-circuit-grey bg-surface-container-low text-xs font-semibold text-on-surface-variant uppercase tracking-wider sticky top-0 z-20 shrink-0">
                        <div class="w-1/12 pr-4">Expand</div>
                        <div class="w-2/12 pr-4">Time / Channel</div>
                        <div class="w-3/12 pr-4">Inbound Message</div>
                        <div class="w-4/12 pr-4">AI Response</div>
                        <div class="w-1/12 pr-4">Latency</div>
                        <div class="w-1/12 text-right">Tokens</div>
                    </div>

                    {/* Body */}
                    <div class="flex-1 relative">
                        {traces.loading && (
                            <div class="absolute inset-0 flex items-center justify-center bg-surface-container-low/50 backdrop-blur-sm z-10">
                                <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-logic-teal"></div>
                            </div>
                        )}
                        {!traces.loading && (
                            <For each={traces()}>
                                {(trace) => (
                                    <div class="border-b border-circuit-grey/50">
                                        {/* Main Row */}
                                        <div 
                                            class={`flex items-center px-6 py-4 cursor-pointer transition-colors ${expandedTraceId() === trace.trace_id ? 'bg-surface-container-highest' : 'hover:bg-surface-container-highest'}`}
                                            onClick={() => toggleExpand(trace.trace_id)}
                                        >
                                            <div class="w-1/12 pr-4 text-terminal-dim">
                                                <svg class={`w-5 h-5 transition-transform ${expandedTraceId() === trace.trace_id ? 'rotate-90 text-logic-teal' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>
                                            </div>
                                            <div class="w-2/12 pr-4 flex flex-col gap-1">
                                                <span class="text-sm font-medium text-on-surface">{new Date(trace.trace_created_at).toLocaleTimeString()}</span>
                                                <div class="flex items-center gap-2">
                                                    <span class="text-xs text-on-surface-variant capitalize">{trace.channel}</span>
                                                    {getHitlBadge(trace.hitl_status || 'none')}
                                                </div>
                                            </div>
                                            <div class="w-3/12 pr-4">
                                                <p class="text-sm text-on-surface truncate pr-4">{trace.inbound_msg}</p>
                                            </div>
                                            <div class="w-4/12 pr-4">
                                                <p class="text-sm text-on-surface-variant line-clamp-2 pr-4">{trace.outbound_msg}</p>
                                            </div>
                                            <div class="w-1/12 pr-4 font-mono text-sm text-on-surface">
                                                {trace.latency_ms}ms
                                            </div>
                                            <div class="w-1/12 text-right font-mono text-sm text-on-surface text-logic-teal">
                                                {trace.tokens_used}
                                            </div>
                                        </div>

                                        {/* Expanded Detail Panel */}
                                        <Show when={expandedTraceId() === trace.trace_id}>
                                            <div class="bg-surface-container p-6 border-t border-circuit-grey/50 shadow-inner">
                                                <div class="grid grid-cols-2 gap-8">
                                                    {/* Left: Chain of Thought */}
                                                    <div>
                                                        <h4 class="text-xs font-bold uppercase tracking-wider text-logic-teal mb-3 flex items-center gap-2">
                                                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
                                                            Agent Thoughts
                                                        </h4>
                                                        <div class="bg-void-black border border-circuit-grey rounded-lg p-4 font-mono text-xs text-on-surface-variant whitespace-pre-wrap max-h-64 overflow-y-auto custom-scrollbar">
                                                            {trace.raw_agent_thoughts || 'No internal thoughts logged.'}
                                                        </div>
                                                    </div>

                                                    {/* Right: Tools & Context */}
                                                    <div class="flex flex-col gap-6">
                                                        <div>
                                                            <h4 class="text-xs font-bold uppercase tracking-wider text-logic-teal mb-3 flex items-center gap-2">
                                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>
                                                                Tools Executed
                                                            </h4>
                                                            <div class="space-y-3 max-h-32 overflow-y-auto custom-scrollbar">
                                                                {(!trace.tools_called || trace.tools_called.length === 0) && (
                                                                    <div class="text-xs text-terminal-dim italic">No tools called in this turn.</div>
                                                                )}
                                                                <For each={trace.tools_called}>
                                                                    {(tool) => (
                                                                        <div class="bg-surface-container-low border border-circuit-grey rounded p-3 text-xs">
                                                                            <div class="font-bold text-on-surface">{tool.name}</div>
                                                                            <div class="font-mono text-terminal-dim mt-1 overflow-x-hidden text-ellipsis whitespace-nowrap">
                                                                                Args: {JSON.stringify(tool.args)}
                                                                            </div>
                                                                        </div>
                                                                    )}
                                                                </For>
                                                            </div>
                                                        </div>

                                                        <div>
                                                            <h4 class="text-xs font-bold uppercase tracking-wider text-logic-teal mb-3 flex items-center gap-2">
                                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
                                                                Retrieved Context
                                                            </h4>
                                                            <div class="space-y-3 max-h-32 overflow-y-auto custom-scrollbar">
                                                                {(!trace.retrieved_contexts || trace.retrieved_contexts.length === 0) && (
                                                                    <div class="text-xs text-terminal-dim italic">No RAG context retrieved.</div>
                                                                )}
                                                                <For each={trace.retrieved_contexts}>
                                                                    {(ctx) => (
                                                                        <div class="bg-surface-container-low border border-circuit-grey/50 rounded p-3 text-xs border-l-2 border-l-logic-teal">
                                                                            <p class="text-on-surface-variant line-clamp-2">{ctx.content}</p>
                                                                            <div class="text-logic-teal font-mono mt-1 opacity-70">Score: {ctx.score}</div>
                                                                        </div>
                                                                    )}
                                                                </For>
                                                            </div>
                                                        </div>
                                                    </div>
                                                </div>
                                            </div>
                                        </Show>
                                    </div>
                                )}
                            </For>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
