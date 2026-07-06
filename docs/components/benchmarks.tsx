"use client";

import { useEffect, useRef, useState } from "react";

const benchmarks = [
    { label: "Plaintext", kern: "87 ns", gin: "72 ns", chi: "129 ns", fiber: "51 ns", kernBest: false, note: "0 allocs" },
    { label: "With Middleware", kern: "100 ns", gin: "116 ns", chi: "154 ns", fiber: "102 ns", kernBest: true, note: "0 allocs" },
    { label: "Query Params", kern: "108 ns", gin: "297 ns", chi: "343 ns", fiber: "98 ns", kernBest: true, note: "0 allocs" },
    { label: "JSON Decode", kern: "546 ns", gin: "664 ns", chi: "680 ns", fiber: "506 ns", kernBest: true, note: "272 B/op" },
    { label: "Path Params", kern: "213 ns", gin: "113 ns", chi: "300 ns", fiber: "122 ns", kernBest: false, note: "48 B/op" },
];

const stats = [
    { value: 0, suffix: "", label: "Dependencies" },
    { value: 87, suffix: "ns", label: "Plaintext Overhead" },
    { value: 100, suffix: "%", label: "Stdlib Compatible" },
    { value: 0, suffix: "", label: "Allocs (plaintext)" },
];

function Counter({ target, suffix, label }: { target: number; suffix: string; label: string }) {
    const [count, setCount] = useState(0);
    const ref = useRef<HTMLDivElement>(null);
    const counted = useRef(false);

    useEffect(() => {
        const el = ref.current;
        if (!el) return;

        const observer = new IntersectionObserver(
            ([entry]) => {
                if (entry.isIntersecting && !counted.current) {
                    counted.current = true;
                    const duration = 1200;
                    const start = performance.now();
                    const animate = (now: number) => {
                        const elapsed = now - start;
                        const progress = Math.min(elapsed / duration, 1);
                        const eased = 1 - Math.pow(1 - progress, 3);
                        setCount(Math.round(target * eased));
                        if (progress < 1) requestAnimationFrame(animate);
                    };
                    requestAnimationFrame(animate);
                }
            },
            { threshold: 0.3 }
        );
        observer.observe(el);
        return () => observer.disconnect();
    }, [target]);

    return (
        <div ref={ref} className="text-center">
            <div className="text-4xl lg:text-5xl font-bold text-white tabular-nums">
                {count}
                <span className="text-2xl lg:text-3xl text-violet-400 ml-1">{suffix}</span>
            </div>
            <div className="text-sm text-slate-500 mt-2">{label}</div>
        </div>
    );
}

export function BenchmarksSection() {
    return (
        <div className="relative py-16 bg-slate-950">
            <div className="max-w-7xl mx-auto px-6 lg:px-8">
                <div className="text-center mb-16">
                    <h2 className="text-3xl lg:text-5xl font-bold mb-4 bg-gradient-to-r from-violet-400 via-indigo-400 to-cyan-400 bg-clip-text text-transparent">
                        Benchmarks you can trust
                    </h2>
                    <p className="text-lg text-slate-400">
                        Apple M3 Pro • Go 1.25 • Lower is better
                    </p>
                </div>

                {/* Quick Stats */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-8 mb-20">
                    {stats.map((stat) => (
                        <Counter key={stat.label} target={stat.value} suffix={stat.suffix} label={stat.label} />
                    ))}
                </div>

                {/* Benchmark Table */}
                <div className="overflow-x-auto rounded-2xl border border-slate-800 bg-slate-900/50 backdrop-blur">
                    <table className="w-full text-left">
                        <thead>
                            <tr className="border-b border-slate-800">
                                <th className="px-6 py-4 text-sm font-semibold text-slate-400">Scenario</th>
                                <th className="px-6 py-4 text-sm font-semibold text-violet-400">Kern</th>
                                <th className="px-6 py-4 text-sm font-semibold text-slate-400">Gin</th>
                                <th className="px-6 py-4 text-sm font-semibold text-slate-400">Chi</th>
                                <th className="px-6 py-4 text-sm font-semibold text-slate-400">Fiber</th>
                                <th className="px-6 py-4 text-sm font-semibold text-slate-400">Note</th>
                            </tr>
                        </thead>
                        <tbody>
                            {benchmarks.map((b) => (
                                <tr key={b.label} className="border-b border-slate-800/50 last:border-0 hover:bg-slate-800/30 transition-colors">
                                    <td className="px-6 py-3.5 text-sm text-slate-300 font-medium">{b.label}</td>
                                    <td className={`px-6 py-3.5 text-sm font-mono font-semibold ${b.kernBest ? "text-emerald-400" : "text-white"}`}>
                                        {b.kern}
                                        {b.kernBest && <span className="ml-1.5 text-xs text-emerald-500">★</span>}
                                    </td>
                                    <td className="px-6 py-3.5 text-sm font-mono text-slate-500">{b.gin}</td>
                                    <td className="px-6 py-3.5 text-sm font-mono text-slate-500">{b.chi}</td>
                                    <td className="px-6 py-3.5 text-sm font-mono text-slate-500">{b.fiber}</td>
                                    <td className="px-6 py-3.5 text-sm text-slate-500">{b.note}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>

                <p className="text-center text-sm text-slate-600 mt-6">
                    Benchmarks run via <code className="text-slate-500">make bench-full</code> •{" "}
                    <a href="https://github.com/mobentum/kern/tree/main/benchmarks/fourway" className="text-violet-400 hover:text-violet-300 transition-colors">
                        View full results on GitHub
                    </a>
                </p>
            </div>
        </div>
    );
}
