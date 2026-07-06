import { getPageImage, source } from '@/lib/source';
import { DocsBody, DocsDescription, DocsPage, DocsTitle } from 'fumadocs-ui/layouts/docs/page';
import { notFound } from 'next/navigation';
import { getMDXComponents } from '@/mdx-components';
import type { Metadata } from 'next';
import { createRelativeLink } from 'fumadocs-ui/mdx';
import { LLMCopyButton, ViewOptions } from '@/components/ai/page-actions';
import { gitConfig } from '@/lib/layout.shared';
import { absoluteUrl, githubUrl, organizationName, siteName } from '@/lib/site';

export default async function Page(props: PageProps<'/docs/[[...slug]]'>) {
  const params = await props.params;
  const page = source.getPage(params.slug);
  if (!page) notFound();

  const MDX = page.data.body;
  const pageUrl = absoluteUrl(page.url);
  const breadcrumbItems = [
    { name: 'Docs', item: absoluteUrl('/docs') },
    ...page.slugs.map((slug, index) => ({
      name: slug.replace(/-/g, ' '),
      item: absoluteUrl(`/docs/${page.slugs.slice(0, index + 1).join('/')}`),
    })),
  ];
  const structuredData = {
    '@context': 'https://schema.org',
    '@graph': [
      {
        '@type': 'TechArticle',
        headline: page.data.title,
        description: page.data.description,
        url: pageUrl,
        about: ['Go web framework', 'net/http routing', 'kern middleware'],
        author: {
          '@type': 'Organization',
          name: organizationName,
        },
        publisher: {
          '@type': 'Organization',
          name: organizationName,
        },
        mainEntityOfPage: pageUrl,
        isPartOf: absoluteUrl('/docs'),
      },
      {
        '@type': 'BreadcrumbList',
        itemListElement: breadcrumbItems.map((item, index) => ({
          '@type': 'ListItem',
          position: index + 1,
          name: item.name,
          item: item.item,
        })),
      },
    ],
  };

  return (
    <DocsPage toc={page.data.toc} full={page.data.full}>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(structuredData) }}
      />
      <DocsTitle>{page.data.title}</DocsTitle>
      <DocsDescription className="mb-0">{page.data.description}</DocsDescription>
      <div className="flex flex-row gap-2 items-center border-b pb-6">
        <LLMCopyButton markdownUrl={`${page.url}.mdx`} />
        <ViewOptions
          markdownUrl={`${page.url}.mdx`}
          githubUrl={`${githubUrl}/blob/${gitConfig.branch}/content/docs/${page.path}`}
        />
      </div>
      <DocsBody>
        <MDX
          components={getMDXComponents({
            // this allows you to link to other pages with relative file paths
            a: createRelativeLink(source, page),
          })}
        />
      </DocsBody>
    </DocsPage>
  );
}

export async function generateStaticParams() {
  return source.generateParams();
}

export async function generateMetadata(props: PageProps<'/docs/[[...slug]]'>): Promise<Metadata> {
  const params = await props.params;
  const page = source.getPage(params.slug);
  if (!page) notFound();

  const description = page.data.description ?? 'kern framework documentation page';
  const image = getPageImage(page).url;

  return {
    title: page.data.title,
    description,
    alternates: {
      canonical: page.url,
    },
    openGraph: {
      type: 'article',
      siteName,
      title: page.data.title,
      description,
      url: page.url,
      images: [
        {
          url: image,
          width: 1200,
          height: 630,
          alt: `${page.data.title} - kern docs`,
        },
      ],
    },
    twitter: {
      card: 'summary_large_image',
      title: page.data.title,
      description,
      images: [image],
    },
  };
}
