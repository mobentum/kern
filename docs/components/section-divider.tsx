export function SectionDivider() {
    return (
        <div className="relative bg-slate-950 h-16 flex items-center justify-center overflow-hidden">
            {/* Background glow */}
            <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,rgba(139,92,246,0.08),transparent_70%)]" />

            {/* Central gradient line */}
            <div className="relative w-full max-w-2xl mx-auto px-8">
                <div className="h-px bg-gradient-to-r from-transparent via-violet-500/30 to-transparent" />
                
                {/* Orbiting dots */}
                <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 flex items-center gap-3">
                    <div className="w-1 h-1 rounded-full bg-violet-400/40" />
                    <div className="w-1.5 h-1.5 rounded-full bg-indigo-400/60 animate-pulse" />
                    <div className="w-1 h-1 rounded-full bg-cyan-400/40" />
                </div>
            </div>
        </div>
    );
}
