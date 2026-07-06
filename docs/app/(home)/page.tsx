import { BenchmarksSection } from "@/components/benchmarks";
import { CodeExampleSection } from "@/components/code-example";
import { ComparisonSection } from "@/components/comparison";
import { CTASection } from "@/components/cta";
import { FeaturesSection } from "@/components/features";
import { HeroSection } from "@/components/hero";
import { ProductionSection } from "@/components/production";
import { SectionDivider } from "@/components/section-divider";
import type { Metadata } from "next";
import { absoluteUrl, githubUrl, organizationName, siteDescription, siteName } from "@/lib/site";

export const metadata: Metadata = {
    title: "Kern — A Fast, Lightweight Go Web Framework",
    description:
        "Kern is a zero-dependency Go HTTP framework with stdlib-native routing, context pooling, built-in middleware, and graceful shutdown — designed for production APIs.",
    alternates: {
        canonical: "/",
    },
};

export default function HomePage() {
    const structuredData = {
        "@context": "https://schema.org",
        "@graph": [
            {
                "@type": "WebPage",
                name: siteName,
                description: siteDescription,
                url: absoluteUrl("/"),
            },
            {
                "@type": "SoftwareSourceCode",
                name: "kern",
                codeRepository: githubUrl,
                programmingLanguage: "Go",
                runtimePlatform: "Go 1.22+",
                description: siteDescription,
                author: {
                    "@type": "Organization",
                    name: organizationName,
                },
            },
        ],
    };

    return (
        <main className="relative overflow-hidden">
            <script
                type="application/ld+json"
                dangerouslySetInnerHTML={{ __html: JSON.stringify(structuredData) }}
            />
            <HeroSection />
            <CodeExampleSection />
            <SectionDivider />
            <FeaturesSection />
            <SectionDivider />
            <ComparisonSection />
            <SectionDivider />
            <BenchmarksSection />
            <SectionDivider />
            <ProductionSection />
            <SectionDivider />
            <CTASection />
        </main>
    );
}
