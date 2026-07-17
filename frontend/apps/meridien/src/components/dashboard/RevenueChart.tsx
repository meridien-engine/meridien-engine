import { createSignal, onMount, For, createResource, createMemo } from 'solid-js';

type RevenuePoint = {
    label: string;
    value: number;
};

// Mock JWT for local development (payload contains business_id: 3b825f22-73a8-487f-90d1-794fdf3e81df)
const DEV_JWT = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJidXNpbmVzc19pZCI6IjNiODI1ZjIyLTczYTgtNDg3Zi05MGQxLTc5NGZkZjNlODFkZiJ9.";

const fetchRevenue = async (): Promise<RevenuePoint[]> => {
    const response = await fetch('http://localhost:8080/api/v1/analytics/revenue', {
        headers: { 'Authorization': `Bearer ${DEV_JWT}` }
    });
    if (!response.ok) throw new Error('Failed to fetch revenue');
    return response.json();
};

export default function RevenueChart() {
    const [animated, setAnimated] = createSignal(false);
    const [revenueData] = createResource(fetchRevenue);

    onMount(() => {
        // Trigger entry animation
        setTimeout(() => setAnimated(true), 100);
    });

    const chartHeight = 200;
    const chartWidth = 800; // SVG internal coordinate width
    
    // Dynamically calculate the maximum value for the Y-axis
    const maxValue = createMemo(() => {
        const data = revenueData() || [];
        if (data.length === 0) return 10000;
        const max = Math.max(...data.map(d => d.value));
        return max > 0 ? max * 1.2 : 10000; // Add 20% padding at the top
    });

    // Generate SVG path coordinates reactively
    const linePath = createMemo(() => {
        const data = revenueData();
        if (!data || data.length === 0) return '';
        
        const stepX = chartWidth / (data.length - 1);
        const max = maxValue();
        
        let path = '';
        data.forEach((point, i) => {
            const x = i * stepX;
            // Map value to Y coordinate (inverted because SVG Y goes down)
            const y = chartHeight - (point.value / max) * chartHeight;
            
            if (i === 0) {
                path += `M ${x},${y} `;
            } else {
                // Smooth bezier curve logic for a premium feel
                const prevX = (i - 1) * stepX;
                const prevY = chartHeight - (data[i - 1].value / max) * chartHeight;
                
                const cp1X = prevX + stepX / 2;
                const cp2X = x - stepX / 2;
                
                path += `C ${cp1X},${prevY} ${cp2X},${y} ${x},${y} `;
            }
        });
        return path;
    });

    // Close the path at the bottom for the gradient fill
    const fillPath = createMemo(() => {
        const p = linePath();
        if (!p) return '';
        return `${p} L ${chartWidth},${chartHeight} L 0,${chartHeight} Z`;
    });

    // Calculate total revenue for the week
    const totalRevenue = createMemo(() => {
        const data = revenueData() || [];
        return data.reduce((sum, item) => sum + item.value, 0);
    });

    return (
        <div class="w-full h-full flex flex-col justify-between pt-2">
            <div class="flex items-center justify-between mb-8">
                <div>
                    <h3 class="text-sm font-medium text-on-surface-variant">Revenue Trajectory (This Week)</h3>
                    <p class="text-3xl font-bold text-on-surface font-heading mt-1 flex items-baseline gap-2">
                        ${totalRevenue().toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                        <span class="text-sm font-medium text-logic-teal flex items-center">
                            <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6"/></svg>
                            Live
                        </span>
                    </p>
                </div>
                <div class="flex gap-2">
                    <button class="px-3 py-1 text-xs font-medium bg-surface-container-high text-on-surface rounded-full border border-circuit-grey">Week</button>
                    <button class="px-3 py-1 text-xs font-medium text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high rounded-full transition-colors">Month</button>
                    <button class="px-3 py-1 text-xs font-medium text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high rounded-full transition-colors">Year</button>
                </div>
            </div>

            <div class="relative flex-1 w-full mt-4">
                {/* Y-Axis Grid Lines */}
                <div class="absolute inset-0 flex flex-col justify-between pointer-events-none">
                    <For each={[1, 0.75, 0.5, 0.25, 0]}>
                        {(multiplier) => (
                            <div class="flex items-center w-full border-b border-circuit-grey/50 border-dashed h-0">
                                <span class="absolute -translate-y-1/2 -left-2 text-[10px] text-terminal-dim font-mono bg-surface-container-low pr-2">
                                    {multiplier === 0 ? '$0' : `$${(maxValue() * multiplier / 1000).toFixed(1)}k`}
                                </span>
                            </div>
                        )}
                    </For>
                </div>

                {/* SVG Chart */}
                <div class="absolute inset-0 pl-8 pb-6">
                    <svg 
                        viewBox={`0 0 ${chartWidth} ${chartHeight}`} 
                        preserveAspectRatio="none" 
                        class="w-full h-full overflow-visible"
                    >
                        <defs>
                            <linearGradient id="revenueGradient" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="0%" stop-color="var(--logic-teal)" stop-opacity="0.3" />
                                <stop offset="100%" stop-color="var(--logic-teal)" stop-opacity="0" />
                            </linearGradient>
                            
                            <filter id="glow" x="-20%" y="-20%" width="140%" height="140%">
                                <feGaussianBlur stdDeviation="4" result="blur" />
                                <feComposite in="SourceGraphic" in2="blur" operator="over" />
                            </filter>
                        </defs>

                        {/* Fill Area */}
                        <path 
                            d={fillPath()} 
                            fill="url(#revenueGradient)"
                            class={`transition-all duration-1000 ease-out origin-bottom ${animated() ? 'opacity-100 scale-y-100' : 'opacity-0 scale-y-0'}`}
                        />
                        
                        {/* Line */}
                        <path 
                            d={linePath()} 
                            fill="none" 
                            stroke="var(--logic-teal)" 
                            stroke-width="3"
                            filter="url(#glow)"
                            class={`transition-all duration-1000 ease-out ${animated() ? 'opacity-100 stroke-dashoffset-0' : 'opacity-0'}`}
                            style="stroke-dasharray: 2000; stroke-dashoffset: 0;"
                        />

                        {/* Data Points */}
                        <For each={revenueData() || []}>
                            {(point, i) => (
                                <circle 
                                    cx={i() * (chartWidth / ((revenueData()?.length || 1) - 1))} 
                                    cy={chartHeight - (point.value / maxValue()) * chartHeight} 
                                    r="4" 
                                    fill="var(--surface-container-lowest)" 
                                    stroke="var(--logic-teal)"
                                    stroke-width="2"
                                    class={`transition-all duration-500 delay-${i() * 100} ${animated() ? 'opacity-100 scale-100' : 'opacity-0 scale-0'}`}
                                />
                            )}
                        </For>
                    </svg>

                    {/* X-Axis Labels */}
                    <div class="absolute bottom-0 left-8 right-0 flex justify-between translate-y-6">
                        <For each={revenueData() || []}>
                            {(point) => (
                                <span class="text-[10px] font-medium text-on-surface-variant uppercase tracking-wider">{point.label}</span>
                            )}
                        </For>
                    </div>
                </div>
            </div>
        </div>
    );
}
