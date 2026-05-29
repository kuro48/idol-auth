import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'idol-auth',
  description: '外部開発者向け認証基盤ドキュメント',
  lang: 'ja',
  base: '/',

  head: [
    ['link', { rel: 'icon', href: '/idol-auth/favicon.ico' }],
  ],

  ignoreDeadLinks: [
    /\/docs\//,
  ],

  themeConfig: {
    siteTitle: 'idol-auth',

    nav: [
      { text: 'ガイド', link: '/guide/' },
      { text: 'API リファレンス', link: '/api/public' },
      { text: 'SDK', link: '/sdk/typescript' },
    ],

    sidebar: {
      '/guide/': [
        {
          text: 'はじめに',
          items: [
            { text: 'クイックスタート', link: '/guide/' },
            { text: 'アプリ申請の流れ', link: '/guide/app-registration' },
          ],
        },
        {
          text: '認証フロー',
          items: [
            { text: 'OIDCフロー（ブラウザ）', link: '/guide/auth-flows' },
            { text: 'Headless 認証（API）', link: '/guide/headless-auth' },
          ],
        },
      ],
      '/api/': [
        {
          text: 'API リファレンス',
          items: [
            { text: 'Public API', link: '/api/public' },
            { text: 'App 管理 API', link: '/api/app-management' },
            { text: 'エラーコード', link: '/api/errors' },
          ],
        },
      ],
      '/sdk/': [
        {
          text: 'SDK',
          items: [
            { text: 'TypeScript / JavaScript', link: '/sdk/typescript' },
          ],
        },
      ],
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/kuro48/idol-auth' },
    ],

    footer: {
      message: 'idol-auth ドキュメント',
    },

    search: {
      provider: 'local',
    },

    outline: {
      label: 'このページの内容',
    },

    docFooter: {
      prev: '前のページ',
      next: '次のページ',
    },

    editLink: {
      pattern: 'https://github.com/kuro48/idol-auth/edit/main/docs-site/:path',
      text: 'このページを GitHub で編集',
    },
  },
})
