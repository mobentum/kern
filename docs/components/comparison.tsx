export function ComparisonSection() {
    return (
        <div className="relative py-16 bg-slate-950">
            <div className="max-w-7xl mx-auto px-6 lg:px-8">
                <div className="text-center mb-16">
                    <h2 className="text-3xl lg:text-5xl font-bold mb-4 bg-gradient-to-r from-violet-400 via-indigo-400 to-cyan-400 bg-clip-text text-transparent">
                        Why Kern over Gin, Fiber, or Chi?
                    </h2>
                    <p className="text-lg text-slate-400">
                        No surprises. No lock-in. Just net/http, done right.
                    </p>
                </div>

                <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
                    {comparisons.map((item) => (
                        <div
                            key={item.title}
                            className="relative bg-gradient-to-br from-slate-900 to-slate-950 rounded-2xl border border-slate-800 p-6 overflow-hidden group hover:border-violet-500/30 transition-all"
                        >
                            <div className="absolute inset-0 bg-gradient-to-br from-violet-500/0 to-violet-500/5 rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity" />
                            <div className="relative">
                                <div className="text-2xl mb-3">{item.icon}</div>
                                <h3 className="text-white font-semibold mb-3 text-lg">{item.title}</h3>
                                <p className="text-slate-400 text-sm leading-relaxed mb-4">{item.description}</p>
                                <div className="space-y-1.5">
                                    {item.pros.map((p) => (
                                        <div key={p} className="flex items-start gap-2 text-xs">
                                            <span className="text-emerald-400 mt-0.5 shrink-0">✓</span>
                                            <span className="text-slate-300">{p}</span>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
}

const comparisons = [
    {
        icon: "🔗",
        title: "Stdlib, Not a Fork",
        description:
            "Kern wraps net/http. Gin and Fiber fork or replace it. Your existing middleware, tooling, and knowledge carry over.",
        pros: [
            "net/http.Handler compatible",
            "Use any stdlib middleware",
            "No custom context type to learn",
        ],
    },
    {
        icon: "📦",
        title: "Zero Dependencies",
        description:
            "Gin pulls 10+ transitive deps. Chi pulls protobuf. Kern's go.mod is empty. Your supply chain stays clean.",
        pros: [
            "Single module, zero deps",
            "No vulnerability churn",
            "Fast CI, tiny binary",
        ],
    },
    {
        icon: "⚡",
        title: "Pooled, Zero-Alloc Hot Paths",
        description:
            "sync.Pool reuses Context objects. Plaintext and middleware paths are allocation-free — Gin and Chi can't say that.",
        pros: [
            "0 allocs on plaintext routes",
            "0 allocs with middleware",
            "Lower GC pressure under load",
        ],
    },
    {
        icon: "🛠️",
        title: "Batteries Included",
        description:
            "JWT, CSRF, sessions, rate limiting, CORS, compression, security headers, ETags, file streaming — all built in.",
        pros: [
            "15+ first-party middleware",
            "File uploads & range streaming",
            "Conditional requests & caching",
        ],
    },
];
