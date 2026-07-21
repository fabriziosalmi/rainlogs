import { defineConfig } from 'vitepress'

export default defineConfig({
  head: [
    // Tutto first-party. 'unsafe-inline' serve perche' VitePress emette
    // uno script inline per il tema e stili inline.
    [
      'meta',
      {
        'http-equiv': 'Content-Security-Policy',
        content:
          "default-src 'self'; script-src 'self' 'unsafe-inline'; " +
          "style-src 'self' 'unsafe-inline'; img-src 'self' data:; " +
          "font-src 'self'; connect-src 'self'; base-uri 'self'; form-action 'self'",
      },
    ],
  ],
  title: "Rainlogs",
  description: "High-performance, self-hosted log management system",
  base: '/rainlogs/',
  themeConfig: {
    logo: '/logo.svg',
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Guide', link: '/guide/introduction' },
      { text: 'API', link: '/guide/api-reference' }
    ],
    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'What is Rainlogs?', link: '/guide/introduction' },
          { text: 'Getting Started', link: '/guide/getting-started' },
          { text: 'Architecture', link: '/guide/architecture' },
        ]
      },
      {
        text: 'Configuration',
        items: [
          { text: 'Environment Variables', link: '/guide/configuration' },
          { text: 'Storage (Garage S3)', link: '/guide/storage' },
        ]
      },
      {
        text: 'Reference',
        items: [
          { text: 'API Reference', link: '/guide/api-reference' },
          { text: 'Deployment', link: '/guide/deployment' },
        ]
      }
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/fabriziosalmi/rainlogs' }
    ],
    footer: {
      message: 'Released under the Apache 2.0 License.' + ' · <a href="https://fabriziosalmi.github.io/privacy">Privacy &amp; legal</a>',
      copyright: 'Copyright © 2026 Fabrizio Salmi'
    },
    search: {
      provider: 'local'
    }
  }
})
