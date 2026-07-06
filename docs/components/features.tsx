const features = [
    {
        icon: "⚡",
        title: "Stdlib-Native Routing",
        description:
            "Built on Go 1.22+ net/http ServeMux with method-based routing, path parameters, and wildcard matching — no custom router to learn or debug.",
        color: "blue",
    },
    {
        icon: "♻️",
        title: "Context Pooling",
        description:
            "sync.Pool-backed Context reuse reduces GC pressure and allocation overhead, keeping your Go API server fast under high concurrency.",
        color: "cyan",
    },
    {
        icon: "🔌",
        title: "Built-in Middleware Suite",
        description:
            "First-party middleware for JWT, CSRF, sessions, rate limiting, CORS, compression, security headers, request guards, and more — all net/http compatible.",
        color: "teal",
    },
    {
        icon: "📦",
        title: "Zero Dependencies",
        description:
            "A single Go module with no external dependencies. Clean go.mod, fast builds, and a minimal supply chain — just import Kern and start building.",
        color: "purple",
    },
    {
        icon: "🛡️",
        title: "Graceful Shutdown",
        description:
            "Built-in OS signal handling with configurable timeouts. Drain in-flight requests cleanly on SIGTERM/SIGINT — no dropped connections in production.",
        color: "pink",
    },
    {
        icon: "🔗",
        title: "Unified Request Binding",
        description:
            "Auto-detect method and Content-Type to bind from query, form, JSON, or XML. Struct tags with built-in validation keep your handlers clean and declarative.",
        color: "green",
    },
    {
        icon: "📁",
        title: "File Upload & Streaming",
        description:
            "Multipart file uploads with SaveFile, byte-range streaming with HTTP 206 Partial Content, and download helpers with Content-Disposition — all built in.",
        color: "yellow",
    },
    {
        icon: "🪝",
        title: "Lifecycle Hooks",
        description:
            "OnRoute, OnListen, OnShutdown, and OnError hooks for observability, metrics, and graceful integration. Structured slog logging with per-request context.",
        color: "red",
    },
    {
        icon: "📖",
        title: "Stdlib-First Design",
        description:
            "A thin layer over net/http with zero custom abstractions. If you know Go's standard library, you already know Kern — onboard your team in minutes.",
        color: "orange",
    },
];

const colorMap: Record<string, { iconBg: string; iconFg: string; border: string; glow: string }> = {
    blue:    { iconBg: "bg-blue-500/10",    iconFg: "text-blue-400",    border: "hover:border-blue-500/50",    glow: "from-blue-500/0 to-blue-500/5" },
    cyan:    { iconBg: "bg-cyan-500/10",    iconFg: "text-cyan-400",    border: "hover:border-cyan-500/50",    glow: "from-cyan-500/0 to-cyan-500/5" },
    teal:    { iconBg: "bg-teal-500/10",    iconFg: "text-teal-400",    border: "hover:border-teal-500/50",    glow: "from-teal-500/0 to-teal-500/5" },
    purple:  { iconBg: "bg-purple-500/10",  iconFg: "text-purple-400",  border: "hover:border-purple-500/50",  glow: "from-purple-500/0 to-purple-500/5" },
    pink:    { iconBg: "bg-pink-500/10",    iconFg: "text-pink-400",    border: "hover:border-pink-500/50",    glow: "from-pink-500/0 to-pink-500/5" },
    green:   { iconBg: "bg-emerald-500/10", iconFg: "text-emerald-400", border: "hover:border-emerald-500/50", glow: "from-emerald-500/0 to-emerald-500/5" },
    yellow:  { iconBg: "bg-amber-500/10",   iconFg: "text-amber-400",   border: "hover:border-amber-500/50",   glow: "from-amber-500/0 to-amber-500/5" },
    red:     { iconBg: "bg-rose-500/10",    iconFg: "text-rose-400",    border: "hover:border-rose-500/50",    glow: "from-rose-500/0 to-rose-500/5" },
    orange:  { iconBg: "bg-orange-500/10",  iconFg: "text-orange-400",  border: "hover:border-orange-500/50",  glow: "from-orange-500/0 to-orange-500/5" },
};

import type { ReactNode } from "react";

const icons: Record<string, ReactNode> = {
    "⚡": (
        <svg className="w-6 h-6 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
        </svg>
    ),
    "♻️": (
        <svg className="w-6 h-6 text-cyan-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
    ),
    "🔌": (
        <svg className="w-6 h-6 text-teal-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
        </svg>
    ),
    "📦": (
        <svg className="w-6 h-6 text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
        </svg>
    ),
    "🛡️": (
        <svg className="w-6 h-6 text-pink-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
        </svg>
    ),
    "🔗": (
        <svg className="w-6 h-6 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
        </svg>
    ),
    "📁": (
        <svg className="w-6 h-6 text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
    ),
    "🪝": (
        <svg className="w-6 h-6 text-rose-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
        </svg>
    ),
    "📖": (
        <svg className="w-6 h-6 text-orange-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
        </svg>
    ),
};

export function FeaturesSection() {
    return (
        <div className="relative py-20 bg-slate-950">
            <div className="max-w-7xl mx-auto px-6 lg:px-8">
                <div className="text-center mb-16">
                    <h2 className="text-3xl lg:text-5xl font-bold mb-4 bg-gradient-to-r from-violet-400 via-indigo-400 to-cyan-400 bg-clip-text text-transparent">
                        A fast, production-ready Go web framework
                    </h2>
                    <p className="text-lg text-slate-400">
                        Stdlib-native routing, zero dependencies, and built-in middleware for high-performance Go APIs
                    </p>
                </div>

                <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-8">
                    {features.map((feature, i) => {
                        const c = colorMap[feature.color];
                        return (
                            <div
                                key={i}
                                className={`group relative bg-gradient-to-br from-slate-900 to-slate-950 p-8 rounded-2xl border border-slate-800 ${c.border} transition-all`}
                            >
                                <div className={`absolute inset-0 bg-gradient-to-br ${c.glow} rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity`} />
                                <div className="relative">
                                    <div className={`w-12 h-12 ${c.iconBg} rounded-lg flex items-center justify-center mb-4`}>
                                        {icons[feature.icon]}
                                    </div>
                                    <h3 className="text-xl font-semibold text-white mb-3">
                                        {feature.title}
                                    </h3>
                                    <p className="text-slate-400 leading-relaxed">
                                        {feature.description}
                                    </p>
                                </div>
                            </div>
                        );
                    })}
                </div>
            </div>
        </div>
    );
}
