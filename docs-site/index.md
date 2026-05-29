---
layout: home

hero:
  name: idol-auth
  text: 共有アカウント型認証基盤
  tagline: 複数アプリが 1 つの ID プールを共有できる OAuth2/OIDC 認証サービスです。
  actions:
    - theme: brand
      text: クイックスタート
      link: /guide/
    - theme: alt
      text: API リファレンス
      link: /api/public

features:
  - icon: 🔐
    title: OAuth2 / OIDC 対応
    details: Ory Hydra をベースにした標準準拠の OAuth2/OIDC サーバー。Authorization Code + PKCE フローでブラウザ・SPA・ネイティブアプリに対応。
  - icon: 👤
    title: 共有アカウントモデル
    details: 1 つのユーザーアカウントで複数アプリにシングルサインオン。アプリごとに独立した OIDC クライアントを発行できます。
  - icon: 📦
    title: TypeScript SDK
    details: '@idol-auth/client パッケージで認証フローを数行で実装。npm install @idol-auth/client でインストール。'
  - icon: ⚡
    title: Headless 認証
    details: ブラウザリダイレクト不要の API モード。モバイルアプリやバックエンドからの直接認証に対応。
---
