import { createSignal } from 'solid-js';

export default function SettingsPanel() {
    const [activeTab, setActiveTab] = createSignal('profile');
    const [ucmEnabled, setUcmEnabled] = createSignal(false);
    const [autoFulfill, setAutoFulfill] = createSignal(true);

    return (
        <div class="flex flex-col lg:flex-row gap-8">
            {/* Sidebar Navigation */}
            <div class="w-full lg:w-64 flex flex-col gap-2 shrink-0">
                <button 
                    onClick={() => setActiveTab('profile')}
                    class={`flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-colors text-left ${activeTab() === 'profile' ? 'bg-surface-container-high text-logic-teal border border-circuit-grey/50' : 'text-on-surface-variant hover:bg-surface-container hover:text-on-surface'}`}
                >
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21v-2a4 4 0 00-4-4H9a4 4 0 00-4 4v2M12 11a4 4 0 100-8 4 4 0 000 8z"/></svg>
                    Business Profile
                </button>
                <button 
                    onClick={() => setActiveTab('ai')}
                    class={`flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-colors text-left ${activeTab() === 'ai' ? 'bg-surface-container-high text-logic-teal border border-circuit-grey/50' : 'text-on-surface-variant hover:bg-surface-container hover:text-on-surface'}`}
                >
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
                    Synapse AI Features
                </button>
                <button 
                    onClick={() => setActiveTab('integrations')}
                    class={`flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-colors text-left ${activeTab() === 'integrations' ? 'bg-surface-container-high text-logic-teal border border-circuit-grey/50' : 'text-on-surface-variant hover:bg-surface-container hover:text-on-surface'}`}
                >
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
                    Integrations & Keys
                </button>
                <button 
                    onClick={() => setActiveTab('billing')}
                    class={`flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-colors text-left ${activeTab() === 'billing' ? 'bg-surface-container-high text-logic-teal border border-circuit-grey/50' : 'text-on-surface-variant hover:bg-surface-container hover:text-on-surface'}`}
                >
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z"/></svg>
                    Billing & Plan
                </button>
            </div>

            {/* Content Area */}
            <div class="flex-1 bg-surface-container-low border border-circuit-grey rounded-2xl p-6 lg:p-8 min-h-[600px]">
                
                {/* ── PROFILE TAB ── */}
                <div class={activeTab() === 'profile' ? 'block' : 'hidden'}>
                    <h3 class="text-xl font-bold text-on-surface mb-6">Business Profile</h3>
                    
                    <div class="space-y-6 max-w-2xl">
                        <div>
                            <label class="block text-xs font-semibold text-on-surface-variant uppercase tracking-wider mb-2">Company Name</label>
                            <input type="text" value="Acme Electronics" class="w-full bg-surface-container border border-circuit-grey rounded-lg px-4 py-3 text-sm text-on-surface focus:outline-none focus:border-logic-teal focus:shadow-[0_0_10px_rgba(194,101,42,0.2)] transition-all" />
                        </div>
                        
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                            <div>
                                <label class="block text-xs font-semibold text-on-surface-variant uppercase tracking-wider mb-2">Support Email</label>
                                <input type="email" value="support@acme.com" class="w-full bg-surface-container border border-circuit-grey rounded-lg px-4 py-3 text-sm text-on-surface focus:outline-none focus:border-logic-teal transition-all" />
                            </div>
                            <div>
                                <label class="block text-xs font-semibold text-on-surface-variant uppercase tracking-wider mb-2">Phone Number</label>
                                <input type="text" value="+1 (555) 123-4567" class="w-full bg-surface-container border border-circuit-grey rounded-lg px-4 py-3 text-sm text-on-surface focus:outline-none focus:border-logic-teal transition-all" />
                            </div>
                        </div>

                        <div>
                            <label class="block text-xs font-semibold text-on-surface-variant uppercase tracking-wider mb-2">Timezone</label>
                            <select class="w-full bg-surface-container border border-circuit-grey rounded-lg px-4 py-3 text-sm text-on-surface focus:outline-none focus:border-logic-teal transition-all appearance-none">
                                <option>UTC - Coordinated Universal Time</option>
                                <option>EST - Eastern Standard Time</option>
                                <option>PST - Pacific Standard Time</option>
                            </select>
                        </div>
                    </div>
                </div>

                {/* ── SYNAPSE AI TAB ── */}
                <div class={activeTab() === 'ai' ? 'block' : 'hidden'}>
                    <h3 class="text-xl font-bold text-on-surface mb-6">Synapse AI Features</h3>
                    
                    <div class="space-y-8 max-w-2xl">
                        {/* Auto-Fulfill Toggle */}
                        <div class="flex items-start justify-between gap-6 p-5 border border-circuit-grey bg-surface-container rounded-xl">
                            <div>
                                <h4 class="text-base font-bold text-on-surface mb-1">Autonomous Order Fulfillment</h4>
                                <p class="text-sm text-on-surface-variant">Allow Synapse AI to automatically process and fulfill standard orders via ERP without human intervention.</p>
                            </div>
                            <button 
                                onClick={() => setAutoFulfill(!autoFulfill())}
                                class={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors shrink-0 ${autoFulfill() ? 'bg-logic-teal' : 'bg-surface-container-highest'}`}
                            >
                                <span class={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${autoFulfill() ? 'translate-x-6' : 'translate-x-1'}`} />
                            </button>
                        </div>

                        {/* Global UCM Toggle (Premium) */}
                        <div class="flex items-start justify-between gap-6 p-6 border border-logic-teal/30 bg-logic-teal/5 rounded-xl relative overflow-hidden">
                            <div class="absolute top-0 right-0 bg-logic-teal text-white text-[9px] font-bold uppercase tracking-widest px-3 py-1 rounded-bl-lg">Premium Tier</div>
                            <div>
                                <h4 class="text-base font-bold text-logic-teal mb-2 flex items-center gap-2">
                                    <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24"><path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/></svg>
                                    Global UCM Network (Opt-In)
                                </h4>
                                <p class="text-sm text-on-surface-variant leading-relaxed">
                                    Join the Unified Customer Model (UCM). By opting in, your system pools anonymized customer interaction vectors with other merchants, dramatically increasing Synapse AI's ability to predict and intercept customer friction before it happens.
                                </p>
                            </div>
                            <button 
                                onClick={() => setUcmEnabled(!ucmEnabled())}
                                class={`relative mt-2 inline-flex h-6 w-11 items-center rounded-full transition-colors shrink-0 ${ucmEnabled() ? 'bg-logic-teal' : 'bg-surface-container-highest'}`}
                            >
                                <span class={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${ucmEnabled() ? 'translate-x-6' : 'translate-x-1'}`} />
                            </button>
                        </div>

                        <div>
                            <label class="block text-xs font-semibold text-on-surface-variant uppercase tracking-wider mb-2">HITL Suspension Threshold (Confidence)</label>
                            <div class="flex items-center gap-4">
                                <input type="range" min="0" max="100" value="85" class="w-full accent-logic-teal" />
                                <span class="text-sm font-mono text-on-surface">85%</span>
                            </div>
                            <p class="text-xs text-terminal-dim mt-2">If the AI's confidence drops below this score, it will automatically route the interaction to the Human Review (HITL) queue.</p>
                        </div>
                    </div>
                </div>

                {/* ── INTEGRATIONS TAB ── */}
                <div class={activeTab() === 'integrations' ? 'block' : 'hidden'}>
                    <h3 class="text-xl font-bold text-on-surface mb-6">Integrations & API</h3>
                    
                    <div class="space-y-6 max-w-2xl">
                        <div class="bg-surface-container border border-circuit-grey rounded-xl p-5">
                            <div class="flex items-center justify-between mb-4">
                                <div class="flex items-center gap-3">
                                    <div class="w-10 h-10 rounded-lg bg-surface-container-highest flex items-center justify-center text-on-surface font-bold text-xl">S</div>
                                    <div>
                                        <h4 class="text-sm font-bold text-on-surface">Shopify / ERP Connection</h4>
                                        <p class="text-xs text-on-surface-variant">Connected & Syncing</p>
                                    </div>
                                </div>
                                <button class="text-xs text-logic-teal font-semibold hover:underline">Disconnect</button>
                            </div>
                            <div class="space-y-3 pt-4 border-t border-circuit-grey/50">
                                <div>
                                    <label class="block text-[10px] uppercase tracking-wider text-on-surface-variant mb-1">Store URL</label>
                                    <input type="text" value="https://acme-electronics.myshopify.com" disabled class="w-full bg-surface-container-low border border-circuit-grey/50 rounded px-3 py-2 text-sm text-terminal-dim opacity-70" />
                                </div>
                                <div>
                                    <label class="block text-[10px] uppercase tracking-wider text-on-surface-variant mb-1">Access Token</label>
                                    <div class="flex gap-2">
                                        <input type="password" value="shpat_1234567890abcdef" disabled class="w-full bg-surface-container-low border border-circuit-grey/50 rounded px-3 py-2 text-sm text-terminal-dim opacity-70 font-mono" />
                                        <button class="px-3 py-2 bg-surface-container-highest rounded border border-circuit-grey hover:bg-surface-container text-xs transition-colors">Rotate</button>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="bg-surface-container border border-circuit-grey rounded-xl p-5">
                            <div class="flex items-center justify-between mb-4">
                                <div class="flex items-center gap-3">
                                    <div class="w-10 h-10 rounded-lg bg-surface-container-highest flex items-center justify-center text-on-surface">
                                        <svg class="w-6 h-6" fill="currentColor" viewBox="0 0 24 24"><path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.888-.788-1.489-1.761-1.663-2.06-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413Z"/></svg>
                                    </div>
                                    <div>
                                        <h4 class="text-sm font-bold text-on-surface">WhatsApp Business API</h4>
                                        <p class="text-xs text-on-surface-variant">Connected to +1 (555) 123-4567</p>
                                    </div>
                                </div>
                                <button class="text-xs text-logic-teal font-semibold hover:underline">Manage</button>
                            </div>
                        </div>
                    </div>
                </div>

                {/* ── BILLING TAB ── */}
                <div class={activeTab() === 'billing' ? 'block' : 'hidden'}>
                    <h3 class="text-xl font-bold text-on-surface mb-6">Billing & Plan</h3>
                    
                    <div class="max-w-2xl bg-gradient-to-br from-logic-teal/10 to-surface-container border border-logic-teal/20 rounded-xl p-6 relative overflow-hidden">
                        <div class="absolute -right-10 -top-10 w-40 h-40 bg-logic-teal/10 rounded-full blur-3xl"></div>
                        <h4 class="text-lg font-bold text-on-surface mb-2">Enterprise Synapse Tier</h4>
                        <p class="text-sm text-on-surface-variant mb-6">You are currently subscribed to the top-tier plan with full AI autonomy, HITL suspension queues, and Global UCM access.</p>
                        
                        <div class="flex items-end gap-2 mb-6">
                            <span class="text-4xl font-bold font-mono text-logic-teal">$499</span>
                            <span class="text-sm text-on-surface-variant pb-1">/ month</span>
                        </div>

                        <button class="bg-surface-container-highest hover:bg-surface-container text-on-surface px-6 py-2.5 rounded-lg text-sm font-medium transition-colors border border-circuit-grey">
                            Manage Subscription
                        </button>
                    </div>
                </div>
                
            </div>
        </div>
    );
}
