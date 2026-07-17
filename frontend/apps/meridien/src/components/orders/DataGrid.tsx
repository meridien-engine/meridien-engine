import { createSignal, createResource, For, createMemo } from 'solid-js';

// Order structure returned by the Go backend REST API
interface Order {
    id: string;
    customerName: string;
    date: string;
    channel: 'whatsapp' | 'messenger' | 'web';
    status: 'pending' | 'completed' | 'cancelled' | 'fulfilled' | 'requires_action' | 'refunded';
    amount: number;
    aiHandled: boolean;
}

// Mock JWT for local development (payload contains business_id: 3b825f22-73a8-487f-90d1-794fdf3e81df)
// The backend middleware currently only parses the payload segment without signature verification in dev.
const DEV_JWT = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJidXNpbmVzc19pZCI6IjNiODI1ZjIyLTczYTgtNDg3Zi05MGQxLTc5NGZkZjNlODFkZiJ9.";

// Fetch orders from the Go backend
const fetchOrders = async (): Promise<Order[]> => {
    const response = await fetch('http://localhost:8080/api/v1/orders?limit=50', {
        headers: {
            'Authorization': `Bearer ${DEV_JWT}`
        }
    });
    if (!response.ok) {
        throw new Error('Failed to fetch orders');
    }
    return response.json();
};

export default function DataGrid() {
    const [searchQuery, setSearchQuery] = createSignal('');
    const [statusFilter, setStatusFilter] = createSignal('All Statuses');
    
    // Fetch data using Solid's createResource
    const [orders, { refetch }] = createResource(fetchOrders);

    // Derived state for filtering
    const filteredOrders = createMemo(() => {
        if (!orders()) return [];
        
        return orders()!.filter(order => {
            const matchesSearch = order.id.toLowerCase().includes(searchQuery().toLowerCase()) || 
                                  order.customerName.toLowerCase().includes(searchQuery().toLowerCase());
            
            const matchesStatus = statusFilter() === 'All Statuses' || order.status.toLowerCase() === statusFilter().toLowerCase();
            
            return matchesSearch && matchesStatus;
        });
    });

    const getStatusStyles = (status: Order['status']) => {
        switch (status) {
            case 'fulfilled': return 'bg-logic-teal/10 text-logic-teal border-logic-teal/30';
            case 'pending': return 'bg-surface-container-high text-on-surface-variant border-circuit-grey';
            case 'requires_action': return 'bg-warning-amber/10 text-warning-amber border-warning-amber/30';
            case 'refunded': return 'bg-circuit-grey text-terminal-dim border-circuit-grey/50';
            case 'completed': return 'bg-logic-teal/10 text-logic-teal border-logic-teal/30';
            case 'cancelled': return 'bg-circuit-grey text-terminal-dim border-circuit-grey/50';
            default: return 'bg-surface-container-high text-on-surface';
        }
    };

    const getStatusLabel = (status: Order['status']) => {
        return status.replace('_', ' ').toUpperCase();
    };

    return (
        <div class="bg-surface-container-low border border-circuit-grey rounded-2xl flex flex-col h-[calc(100vh-160px)]">
            
            {/* Toolbar */}
            <div class="p-5 border-b border-circuit-grey flex items-center justify-between gap-4">
                <div class="flex items-center gap-4 flex-1">
                    <div class="relative w-80">
                        <svg class="w-4 h-4 text-terminal-dim absolute left-4 top-1/2 -translate-y-1/2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>
                        <input 
                            type="text" 
                            placeholder="Search orders by ID or customer..." 
                            value={searchQuery()}
                            onInput={(e) => setSearchQuery(e.currentTarget.value)}
                            class="w-full bg-surface-container-lowest border border-circuit-grey rounded-full py-2 pl-10 pr-4 text-sm text-on-surface focus:outline-none focus:border-logic-teal transition-colors"
                        />
                    </div>
                    <select 
                        value={statusFilter()}
                        onChange={(e) => setStatusFilter(e.currentTarget.value)}
                        class="bg-surface-container-lowest border border-circuit-grey rounded-full py-2 px-4 pr-8 text-sm text-on-surface appearance-none focus:outline-none focus:border-logic-teal transition-colors cursor-pointer"
                    >
                        <option value="All Statuses">All Statuses</option>
                        <option value="requires_action">Requires Action</option>
                        <option value="pending">Pending</option>
                        <option value="fulfilled">Fulfilled</option>
                    </select>
                </div>

                <button class="bg-logic-teal hover:bg-logic-teal/90 text-white font-medium text-sm py-2 px-6 rounded-full transition-colors flex items-center gap-2">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>
                    Create Order
                </button>
            </div>

            {/* Table Header */}
            <div class="flex items-center px-6 py-4 border-b border-circuit-grey bg-surface-container-low text-xs font-semibold text-on-surface-variant uppercase tracking-wider">
                <div class="w-2/12 pr-4">Order ID</div>
                <div class="w-3/12 pr-4">Customer</div>
                <div class="w-2/12 pr-4">Date</div>
                <div class="w-2/12 pr-4">Channel</div>
                <div class="w-2/12 pr-4">Status</div>
                <div class="w-1/12 text-right">Amount</div>
            </div>

            {/* Table Body */}
            <div class="flex-1 overflow-y-auto relative">
                {orders.loading && (
                    <div class="absolute inset-0 flex items-center justify-center bg-surface-container-low/50 backdrop-blur-sm z-10">
                        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-logic-teal"></div>
                    </div>
                )}
                {orders.error && (
                    <div class="p-6 text-center text-warning-amber text-sm font-medium">
                        Failed to load orders. Make sure the Go backend is running.
                    </div>
                )}
                {!orders.loading && !orders.error && (
                    <For each={filteredOrders()}>
                        {(order) => (
                            <div class="flex items-center px-6 py-4 border-b border-circuit-grey/50 hover:bg-surface-container-highest transition-colors group cursor-pointer">
                                <div class="w-2/12 pr-4 font-mono text-sm font-medium text-on-surface">
                                    {order.id}
                                </div>
                                <div class="w-3/12 pr-4 flex items-center gap-3">
                                    <div class="w-8 h-8 rounded-full bg-logic-teal/10 flex-shrink-0 flex items-center justify-center text-logic-teal text-xs font-bold uppercase">
                                        {order.customerName.charAt(0)}
                                    </div>
                                    <span class="text-sm font-medium text-on-surface truncate">{order.customerName}</span>
                                </div>
                                <div class="w-2/12 pr-4 text-sm text-on-surface-variant">
                                    {new Date(order.date).toLocaleDateString()}
                                </div>
                                <div class="w-2/12 pr-4 flex items-center gap-2">
                                    <span class="text-xs text-on-surface-variant capitalize">{order.channel}</span>
                                    {order.aiHandled && (
                                        <span class="w-4 h-4 bg-logic-teal/10 rounded-full flex-shrink-0 flex items-center justify-center" title="Handled by AI">
                                            <svg class="w-2.5 h-2.5 text-logic-teal" fill="currentColor" viewBox="0 0 24 24"><path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/></svg>
                                        </span>
                                    )}
                                </div>
                                <div class="w-2/12 pr-4">
                                    <span class={`text-[10px] font-bold px-3 py-1.5 rounded-full border whitespace-nowrap inline-flex items-center justify-center ${getStatusStyles(order.status)}`}>
                                        {getStatusLabel(order.status)}
                                    </span>
                                </div>
                                <div class="w-1/12 text-right font-mono text-sm text-on-surface">
                                    ${order.amount.toFixed(2)}
                                </div>
                            </div>
                        )}
                    </For>
                )}
                {(!orders.loading && !orders.error && filteredOrders().length === 0) && (
                    <div class="flex flex-col items-center justify-center h-full text-terminal-dim py-12">
                        <svg class="w-12 h-12 mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>
                        <p class="text-sm">No orders found matching your criteria.</p>
                    </div>
                )}
            </div>
            
            {/* Pagination / Footer */}
            <div class="p-4 border-t border-circuit-grey flex items-center justify-between text-sm text-on-surface-variant bg-surface-container-lowest rounded-b-2xl">
                <span>Showing {filteredOrders().length} of {orders()?.length || 0} orders</span>
                <div class="flex items-center gap-4">
                    <button class="hover:text-logic-teal transition-colors flex items-center gap-1" onClick={refetch}>
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
                        Refresh
                    </button>
                    <div class="w-px h-4 bg-circuit-grey"></div>
                    <button class="p-1 hover:text-logic-teal disabled:opacity-50" disabled><svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/></svg></button>
                    <button class="p-1 hover:text-logic-teal disabled:opacity-50" disabled><svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg></button>
                </div>
            </div>

        </div>
    );
}
