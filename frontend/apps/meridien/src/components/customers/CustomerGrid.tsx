import { createResource, For } from 'solid-js';

// Mock JWT for local development (payload contains business_id: 3b825f22-73a8-487f-90d1-794fdf3e81df)
const DEV_JWT = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJidXNpbmVzc19pZCI6IjNiODI1ZjIyLTczYTgtNDg3Zi05MGQxLTc5NGZkZjNlODFkZiJ9.";

type CustomerResponse = {
    id: string;
    name: string;
    tier: string;
    summary: string;
    primary_channel: string;
    joined_at: string;
};

const fetchCustomers = async (): Promise<CustomerResponse[]> => {
    const response = await fetch('http://localhost:8080/api/v1/customers', {
        headers: { 'Authorization': `Bearer ${DEV_JWT}` }
    });
    if (!response.ok) throw new Error('Failed to fetch customers');
    return response.json();
};

export default function CustomerGrid() {
    const [customers] = createResource(fetchCustomers);

    const getTierBadge = (tier: string) => {
        switch (tier.toLowerCase()) {
            case 'gold':
                return <span class="px-2 py-1 text-[10px] font-medium rounded-full bg-yellow-500/10 text-yellow-500 border border-yellow-500/20">Gold</span>;
            case 'silver':
                return <span class="px-2 py-1 text-[10px] font-medium rounded-full bg-slate-400/10 text-slate-400 border border-slate-400/20">Silver</span>;
            default:
                return <span class="px-2 py-1 text-[10px] font-medium rounded-full bg-circuit-grey/20 text-on-surface-variant border border-circuit-grey">Standard</span>;
        }
    };

    const getChannelIcon = (channel: string) => {
        switch (channel?.toLowerCase()) {
            case 'whatsapp':
                return <span class="text-green-500 text-xs flex items-center gap-1"><svg class="w-3 h-3" fill="currentColor" viewBox="0 0 24 24"><path d="M12.031 6.172c-3.181 0-5.767 2.586-5.768 5.766-.001 1.298.38 2.27 1.019 3.287l-.582 2.128 2.182-.573c.978.58 1.911.928 3.145.929 3.178 0 5.767-2.587 5.768-5.766.001-3.187-2.575-5.77-5.764-5.771zm3.392 8.244c-.144.405-.837.774-1.17.824-.299.045-.677.063-1.092-.069-.252-.08-.575-.187-.988-.365-1.739-.751-2.874-2.502-2.961-2.617-.087-.116-.708-.94-.708-1.793s.448-1.273.607-1.446c.159-.173.346-.217.462-.217l.332.006c.106.005.249-.04.39.298.144.347.491 1.2.534 1.287.043.087.072.188.014.304-.058.116-.087.188-.173.289l-.26.304c-.087.086-.177.18-.076.354.101.174.449.741.964 1.201.662.591 1.221.774 1.394.86s.274.072.376-.043c.101-.116.433-.506.549-.68.116-.173.231-.145.39-.087s1.011.477 1.184.564.289.13.332.202c.045.072.045.419-.099.824z"/></svg>WhatsApp</span>;
            case 'messenger':
                return <span class="text-blue-500 text-xs flex items-center gap-1"><svg class="w-3 h-3" fill="currentColor" viewBox="0 0 24 24"><path d="M12 2C6.477 2 2 6.145 2 11.259c0 2.87 1.455 5.434 3.738 7.127l-.608 2.213a.5.5 0 00.627.606l2.45-.662A9.771 9.771 0 0012 20.518c5.523 0 10-4.145 10-9.259S17.523 2 12 2zm1.196 12.392l-2.483-2.656-4.814 2.656 5.297-5.632 2.502 2.656 4.795-2.656-5.297 5.632z"/></svg>Messenger</span>;
            default:
                return <span class="text-terminal-dim text-xs flex items-center gap-1">Web</span>;
        }
    };

    return (
        <div class="bg-surface-container-low border border-circuit-grey rounded-2xl overflow-hidden">
            <div class="overflow-x-auto custom-scrollbar">
                <table class="w-full min-w-[900px] text-left text-sm text-on-surface">
                    <thead class="bg-surface-container uppercase text-[10px] tracking-wider text-on-surface-variant font-medium border-b border-circuit-grey">
                        <tr>
                            <th scope="col" class="px-6 py-4">Customer Name</th>
                            <th scope="col" class="px-6 py-4">Tier</th>
                            <th scope="col" class="px-6 py-4">Primary Channel</th>
                            <th scope="col" class="px-6 py-4 w-[40%]">AI Semantic Summary</th>
                            <th scope="col" class="px-6 py-4">Joined</th>
                            <th scope="col" class="px-6 py-4 text-right">Actions</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-circuit-grey/50">
                        <For each={customers()} fallback={
                            <tr><td colspan="6" class="px-6 py-8 text-center text-terminal-dim font-mono">Loading customer profiles...</td></tr>
                        }>
                            {(customer) => (
                                <tr class="hover:bg-logic-teal/5 transition-colors group">
                                    <td class="px-6 py-4">
                                        <div class="font-medium text-on-surface">{customer.name || 'Anonymous User'}</div>
                                        <div class="text-xs text-terminal-dim font-mono mt-0.5">{customer.id.substring(0, 8)}...</div>
                                    </td>
                                    <td class="px-6 py-4">
                                        {getTierBadge(customer.tier)}
                                    </td>
                                    <td class="px-6 py-4">
                                        {getChannelIcon(customer.primary_channel)}
                                    </td>
                                    <td class="px-6 py-4">
                                        <p class="text-xs text-on-surface-variant line-clamp-2 leading-relaxed">
                                            {customer.summary || 'No AI context gathered yet. Waiting for interaction.'}
                                        </p>
                                    </td>
                                    <td class="px-6 py-4 text-xs text-terminal-dim">
                                        {new Date(customer.joined_at).toLocaleDateString()}
                                    </td>
                                    <td class="px-6 py-4 text-right">
                                        <button class="text-logic-teal hover:text-white transition-colors opacity-0 group-hover:opacity-100 p-1.5 rounded bg-logic-teal/10 hover:bg-logic-teal">
                                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>
                                        </button>
                                    </td>
                                </tr>
                            )}
                        </For>
                    </tbody>
                </table>
            </div>
        </div>
    );
}
