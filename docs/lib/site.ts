export const siteName = "kern Docs";
export const siteTitle = "kern Docs - Lightweight Go Web Framework";
export const siteDescription =
  "Official kern documentation for building fast, minimal, and production-ready Go web services with stdlib-native routing and middleware.";
export const organizationName = "mobentum";
export const githubUrl = "https://github.com/mobentum/kern";

export function getSiteUrl(): string {
  const configured =
    process.env.NEXT_PUBLIC_SITE_URL ??
    process.env.VERCEL_PROJECT_PRODUCTION_URL ??
    "http://localhost:3000";

  return configured.startsWith("http") ? configured : `https://${configured}`;
}

export function absoluteUrl(path: string): string {
  const normalized = path.startsWith("/") ? path : `/${path}`;
  return new URL(normalized, getSiteUrl()).toString();
}
