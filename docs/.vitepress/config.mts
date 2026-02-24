import { defineConfig } from 'vitepress'

export default defineConfig({
  title: "Rainlogs",
  description: "High-performance, self-hosted log management system",
  base: '/rainlogs/',
  themeConfig: {
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
      message: 'Released under the Apache 2.0 License.',
      copyright: 'Copyright © 2026 Fabrizio Salmi'
    },
    search: {
      provider: 'local'
    }
  }
})
