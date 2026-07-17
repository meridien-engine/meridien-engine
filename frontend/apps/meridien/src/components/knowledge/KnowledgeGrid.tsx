import { createResource, createSignal, For, Show } from 'solid-js';

const DEV_JWT = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJidXNpbmVzc19pZCI6IjNiODI1ZjIyLTczYTgtNDg3Zi05MGQxLTc5NGZkZjNlODFkZiJ9.";

type KnowledgeNode = {
    id: string;
    source_name: string;
    preview: string;
    created_at: string;
};

const fetchKnowledge = async (): Promise<KnowledgeNode[]> => {
    const response = await fetch('http://localhost:8080/api/v1/knowledge', {
        headers: { 'Authorization': `Bearer ${DEV_JWT}` }
    });
    if (!response.ok) throw new Error('Failed to fetch knowledge base');
    return response.json();
};

export default function KnowledgeGrid() {
    const [knowledge, { refetch }] = createResource(fetchKnowledge);
    const [isSubmitting, setIsSubmitting] = createSignal(false);
    const [showModal, setShowModal] = createSignal(false);

    const [sourceName, setSourceName] = createSignal('');
    const [content, setContent] = createSignal('');

    const handleSubmit = async (e: Event) => {
        e.preventDefault();
        if (!sourceName() || !content()) return;

        setIsSubmitting(true);
        try {
            const res = await fetch('http://localhost:8080/api/v1/knowledge', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${DEV_JWT}`
                },
                body: JSON.stringify({
                    source_name: sourceName(),
                    content: content()
                })
            });

            if (res.ok) {
                setSourceName('');
                setContent('');
                setShowModal(false);
                refetch();
            }
        } catch (error) {
            console.error("Failed to add knowledge node:", error);
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <div>
            {/* Topbar Actions */}
            <div class="flex justify-between items-center mb-6">
                <p class="text-sm text-on-surface-variant flex items-center gap-2">
                    <svg class="w-4 h-4 text-logic-teal" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
                    The AI uses these documents as ground-truth when answering customer queries.
                </p>
                <button 
                    onClick={() => setShowModal(true)}
                    class="bg-logic-teal hover:bg-logic-teal/90 text-surface-container-lowest px-4 py-2 rounded-lg text-sm font-medium transition-colors shadow-lg shadow-logic-teal/20 flex items-center gap-2"
                >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>
                    Add Knowledge
                </button>
            </div>

            {/* Grid */}
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                <For each={knowledge()} fallback={
                    <div class="col-span-3 py-12 text-center border-2 border-dashed border-circuit-grey rounded-2xl">
                        <p class="text-terminal-dim font-mono text-sm">No knowledge nodes found. Add one to train the AI.</p>
                    </div>
                }>
                    {(node) => (
                        <div class="bg-surface-container-low border border-circuit-grey hover:border-logic-teal/50 rounded-2xl p-6 transition-all group flex flex-col h-64 relative overflow-hidden">
                            <div class="absolute inset-0 bg-logic-teal/5 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"></div>
                            <div class="flex justify-between items-start mb-4 relative">
                                <div class="flex items-center gap-2">
                                    <div class="w-8 h-8 rounded-lg bg-surface-container-high border border-circuit-grey flex items-center justify-center shrink-0">
                                        <svg class="w-4 h-4 text-logic-teal" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>
                                    </div>
                                    <h3 class="font-medium text-on-surface text-sm line-clamp-2 leading-tight">{node.source_name}</h3>
                                </div>
                            </div>
                            
                            <div class="flex-1 bg-void-black/50 border border-circuit-grey/50 rounded-xl p-3 overflow-hidden relative group-hover:border-logic-teal/30 transition-colors">
                                <p class="text-xs text-terminal-dim font-mono leading-relaxed whitespace-pre-wrap break-words">
                                    {node.preview.length >= 200 ? node.preview + '...' : node.preview}
                                </p>
                                <div class="absolute inset-x-0 bottom-0 h-12 bg-gradient-to-t from-void-black/80 to-transparent"></div>
                            </div>

                            <div class="mt-4 pt-4 border-t border-circuit-grey/50 flex justify-between items-center text-[10px] text-on-surface-variant font-medium relative">
                                <span class="flex items-center gap-1">
                                    <svg class="w-3 h-3 text-logic-teal" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
                                    Vectorized
                                </span>
                                <span>{new Date(node.created_at).toLocaleDateString()}</span>
                            </div>
                        </div>
                    )}
                </For>
            </div>

            {/* Modal */}
            <Show when={showModal()}>
                <div class="fixed inset-0 z-50 flex items-center justify-center bg-void-black/80 backdrop-blur-sm">
                    <div class="bg-surface-container-low border border-circuit-grey rounded-2xl w-full max-w-lg p-6 shadow-2xl relative animate-in fade-in zoom-in-95">
                        <button onClick={() => setShowModal(false)} class="absolute top-4 right-4 text-on-surface-variant hover:text-white">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
                        </button>
                        
                        <h2 class="text-xl font-heading font-bold text-on-surface mb-6 flex items-center gap-2">
                            <svg class="w-5 h-5 text-logic-teal" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
                            Inject AI Knowledge
                        </h2>

                        <form onSubmit={handleSubmit} class="space-y-4">
                            <div>
                                <label class="block text-xs font-medium text-on-surface-variant mb-1.5 uppercase tracking-wider">Source Name (e.g. Return Policy)</label>
                                <input 
                                    type="text" 
                                    required
                                    value={sourceName()}
                                    onInput={(e) => setSourceName(e.currentTarget.value)}
                                    class="w-full bg-void-black border border-circuit-grey rounded-xl px-4 py-2.5 text-sm text-on-surface focus:outline-none focus:border-logic-teal focus:ring-1 focus:ring-logic-teal transition-all placeholder:text-terminal-dim"
                                    placeholder="Enter document title..."
                                />
                            </div>
                            <div>
                                <label class="block text-xs font-medium text-on-surface-variant mb-1.5 uppercase tracking-wider">Raw Content (will be vectorized)</label>
                                <textarea 
                                    required
                                    value={content()}
                                    onInput={(e) => setContent(e.currentTarget.value)}
                                    rows="6"
                                    class="w-full bg-void-black border border-circuit-grey rounded-xl px-4 py-2.5 text-sm text-on-surface font-mono focus:outline-none focus:border-logic-teal focus:ring-1 focus:ring-logic-teal transition-all placeholder:text-terminal-dim custom-scrollbar resize-none"
                                    placeholder="Enter the text context you want the AI to memorize..."
                                ></textarea>
                            </div>
                            
                            <div class="pt-4 flex justify-end gap-3 border-t border-circuit-grey/50">
                                <button 
                                    type="button" 
                                    onClick={() => setShowModal(false)}
                                    class="px-4 py-2 text-sm font-medium text-on-surface-variant hover:text-white transition-colors"
                                >
                                    Cancel
                                </button>
                                <button 
                                    type="submit" 
                                    disabled={isSubmitting()}
                                    class="bg-logic-teal hover:bg-logic-teal/90 disabled:opacity-50 disabled:cursor-not-allowed text-surface-container-lowest px-6 py-2 rounded-lg text-sm font-medium transition-all shadow-lg shadow-logic-teal/20 flex items-center gap-2"
                                >
                                    {isSubmitting() ? 'Vectorizing...' : 'Save to Brain'}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            </Show>
        </div>
    );
}
