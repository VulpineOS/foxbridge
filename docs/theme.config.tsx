import type { DocsThemeConfig } from 'nextra-theme-docs'

const config: DocsThemeConfig = {
  logo: <span style={{ fontWeight: 800 }}>🦊 Foxbridge</span>,
  project: {
    link: 'https://github.com/PopcornDev1/foxbridge',
  },
  docsRepositoryBase: 'https://github.com/PopcornDev1/foxbridge/tree/main/docs',
  footer: {
    content: 'Foxbridge — CDP-to-Firefox Protocol Proxy',
  },
  head: (
    <>
      <meta name="viewport" content="width=device-width, initial-scale=1.0" />
      <meta name="description" content="Foxbridge documentation — translate Chrome DevTools Protocol to Firefox Juggler and WebDriver BiDi" />
      <title>Foxbridge Docs</title>
    </>
  ),
  sidebar: {
    defaultMenuCollapseLevel: 1,
  },
  banner: {
    key: 'vulpineos',
    content: (
      <a href="https://vulpineos.com" target="_blank" rel="noopener">
        Part of the VulpineOS ecosystem → Learn more about VulpineOS
      </a>
    ),
  },
}

export default config
