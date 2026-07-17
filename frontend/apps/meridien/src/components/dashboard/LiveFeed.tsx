import { createSignal, onMount, onCleanup, For, createResource } from 'solid-js';

type FeedItem = {
    id: string;
    type: 'success' | 'info' | 'warning';
    title: string;
    description: string;
    timestamp: Date;
    meta?: string;
};

// Mock JWT for local development (payload contains business_id: 3b825f22-73a8-487f-90d1-794fdf3e81df)
const DEV_JWT = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJidXNpbmVzc19pZCI6IjNiODI1ZjIyLTczYTgtNDg3Zi05MGQxLTc5NGZkZjNlODFkZiJ9.";

const fetchFeed = async (): Promise<FeedItem[]> => {
    const response = await fetch('/api/v1/feed', {
        headers: { 'Authorization': `Bearer ${DEV_JWT}` }
    });
    if (!response.ok) throw new Error('Failed to fetch feed');
    
    const data = await response.json();
    return data.map((item: any) => ({
        ...item,
        timestamp: new Date(item.timestamp) // Parse ISO string to Date object
    }));
};

export default function LiveFeed() {
    const [feed, { refetch }] = createResource(fetchFeed);

    // Poll the backend every 3 seconds for new live events
    onMount(() => {
        const interval = setInterval(refetch, 3000);
        onCleanup(() => clearInterval(interval));
    });

    const formatTimeAgo = (date: Date) => {
        const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);
        if (seconds < 60) return 'Just now';
        return `${Math.floor(seconds / 60)}m ago`;
    };

    return (
        <div class="h-full flex flex-col">
            <h3 class="font-medium text-sm text-logic-teal mb-6 pb-3 border-b border-circuit-grey flex items-center justify-between">
                Live Feed
                <span class="text-[10px] px-2 py-0.5 border border-logic-teal text-logic-teal rounded-full animate-pulse font-medium">LIVE</span>
            </h3>

            <div class="flex-1 overflow-y-auto space-y-4 pr-2 custom-scrollbar">
                <For each={feed()}>
                    {(item) => (
                        <div 
                            class={`flex gap-4 items-start border-l-2 pl-3 transition-all duration-300 animate-in fade-in slide-in-from-top-4 ${
                                item.type === 'warning' 
                                    ? 'border-warning-amber bg-warning-amber/5 py-2 rounded-r-lg' 
                                    : 'border-logic-teal/30 hover:border-logic-teal'
                            }`}
                        >
                            <div class="flex-1 min-w-0">
                                <div class="flex justify-between items-start mb-1">
                                    <p class={`text-sm font-medium truncate pr-2 ${
                                        item.type === 'warning' ? 'text-warning-amber' : 'text-on-surface'
                                    }`}>
                                        {item.title}
                                    </p>
                                    {item.meta && (
                                        <span class={`text-[10px] flex-shrink-0 mt-0.5 ${
                                            item.type === 'warning' ? 'text-warning-amber' : 'text-logic-teal'
                                        }`}>
                                            {item.meta}
                                        </span>
                                    )}
                                </div>
                                <p class="text-xs text-on-surface-variant line-clamp-2">{item.description}</p>
                                <p class={`text-[10px] mt-1.5 ${
                                    item.type === 'warning' ? 'text-warning-amber/70' : 'text-terminal-dim'
                                }`}>
                                    {formatTimeAgo(item.timestamp)}
                                </p>
                            </div>
                        </div>
                    )}
                </For>
            </div>
        </div>
    );
}
