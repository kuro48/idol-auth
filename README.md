# idol-auth

複数サービスで共通のユーザーアカウントを利用するための認証基盤です。

Ory Kratos（アイデンティティ管理）と Ory Hydra（OAuth2/OIDC プロバイダー）を組み合わせ、Authorization Code + PKCE フローによるシングルサインオンを提供します。外部開発者は OAuth2 クライアントを登録するだけで、idol-auth のアカウントシステムを自分のアプリに組み込めます。

## ドキュメント

外部開発者向けのドキュメントはアプリ内に組み込まれています。

| ページ | 内容 |
|---|---|
| [概要](/docs) | idol-auth の全体像と外部開発者が扱う範囲 |
| [最初の連携を作る](/docs/start) | アプリ登録からコールバック処理まで |
| [TypeScript SDK](/docs/sdk) | `@idol-auth/client` の使い方 |
| [API Reference](/docs/api) | 全エンドポイントの詳細仕様 |
| [概念と用語](/docs/concepts) | OAuth2 / OIDC の基本概念 |
| [ユーザー管理](/docs/management) | Management Token を使ったユーザー操作 |
| [セキュリティチェックリスト](/docs/security) | 本番前に確認すべき実装ポイント |

## 構成

```
backend/    Go製APIサーバー（Ory Kratos / Hydra と連携）
frontend/   React製フロントエンド（ログイン・開発者ポータル・ドキュメント）
sdk/        TypeScript SDK (@idol-auth/client)
```

## ローカル開発

```bash
# 依存サービスを起動（Postgres / Kratos / Hydra）
make up

# フロントエンド開発サーバーを起動
make frontend-dev

# バックエンドのテストを実行
make test

# E2Eテストを実行
make e2e
```

起動後、`http://localhost:3000` でアプリにアクセスできます。ドキュメントは `http://localhost:3000/docs` から参照してください。

## ライセンス

[LICENSE](LICENSE) を参照してください。
