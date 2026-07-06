import { RootProvider } from 'fumadocs-ui/provider/next';
import './global.css';
import { Inter } from 'next/font/google';
import type { Metadata, Viewport } from 'next';
import { getSiteUrl, siteDescription, siteName, siteTitle, organizationName, githubUrl } from '@/lib/site';

const inter = Inter({
  subsets: ['latin'],
});

export const metadata: Metadata = {
  metadataBase: new URL(getSiteUrl()),
  title: {
    default: siteTitle,
    template: `%s | ${siteName}`,
  },
  description: siteDescription,
  applicationName: 'kern',
  keywords: [
    'Go web framework',
    'Go router',
    'kern',
    'net/http',
    'middleware',
    'API framework',
    'Golang documentation',
  ],
  authors: [{ name: 'mobentum' }],
  creator: 'mobentum',
  publisher: 'mobentum',
  alternates: {
    canonical: '/',
  },
  openGraph: {
    type: 'website',
    locale: 'en_US',
    siteName,
    title: siteTitle,
    description: siteDescription,
    url: '/',
    images: [
      {
        url: '/branding/kern-logo-stacked.svg',
        width: 900,
        height: 900,
        alt: 'kern colorful logo',
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    title: siteTitle,
    description: siteDescription,
    images: ['/branding/kern-logo-stacked.svg'],
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      'max-image-preview': 'large',
      'max-snippet': -1,
      'max-video-preview': -1,
    },
  },
};

export const viewport: Viewport = {
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#ecfeff' },
    { media: '(prefers-color-scheme: dark)', color: '#0f172a' },
  ],
};

export default function Layout({ children }: LayoutProps<'/'>) {
  const siteUrl = getSiteUrl();
  const structuredData = {
    '@context': 'https://schema.org',
    '@graph': [
      {
        '@type': 'Organization',
        '@id': `${siteUrl}#organization`,
        name: organizationName,
        url: siteUrl,
        logo: `${siteUrl}/branding/kern-logo-stacked.svg`,
        sameAs: [githubUrl],
      },
      {
        '@type': 'WebSite',
        '@id': `${siteUrl}#website`,
        url: siteUrl,
        name: siteName,
        description: siteDescription,
        publisher: {
          '@id': `${siteUrl}#organization`,
        },
        potentialAction: {
          '@type': 'SearchAction',
          target: `${siteUrl}/api/search?q={search_term_string}`,
          'query-input': 'required name=search_term_string',
        },
      },
    ],
  };

  return (
    <html lang="en" className={inter.className} suppressHydrationWarning>
      <body className="flex flex-col min-h-screen">
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(structuredData) }}
        />
        <RootProvider>{children}</RootProvider>
      </body>
    </html>
  );
}
