import { generateStaticParamsFor, importPage } from 'nextra/pages'
import { useMDXComponents as getMDXComponents } from '../../mdx-components'

export const generateStaticParams = generateStaticParamsFor('mdxPath')

export async function generateMetadata(props) {
  const params = await props.params
  const { metadata } = await importPage(params.mdxPath)
  const title = metadata?.title || 'Foxbridge'
  const description = metadata?.description || 'CDP-to-Firefox Protocol Proxy'
  const ogUrl = `/api/og?title=${encodeURIComponent(title)}&description=${encodeURIComponent(description)}`
  return {
    ...metadata,
    openGraph: {
      title,
      description,
      images: [{ url: ogUrl, width: 1200, height: 630, alt: title }],
    },
    twitter: {
      card: 'summary_large_image',
      title,
      description,
      images: [ogUrl],
    },
  }
}

const Wrapper = getMDXComponents().wrapper

export default async function Page(props) {
  const params = await props.params
  const { default: MDXContent, toc, metadata } = await importPage(params.mdxPath)
  return (
    <Wrapper toc={toc} metadata={metadata}>
      <MDXContent {...props} params={params} />
    </Wrapper>
  )
}
