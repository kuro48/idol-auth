# idol-auth Docs

このディレクトリは idol-auth の開発・運用ドキュメントです。実行中の API サーバーでは Swagger UI も配信します。

## Docs Site

ローカル起動後に次を開きます。

| 用途 | URL |
|---|---|
| Docs entry | `http://localhost:8080/docs` |
| Swagger UI | `http://localhost:8080/docs/index.html` |
| Swagger JSON | `http://localhost:8080/docs/doc.json` |

`/docs` と `/docs/` は Swagger UI の `index.html` へリダイレクトされます。

## 読む順番

1. [連携ガイド](integration.md): アプリ開発者向け。アプリ登録、OIDC クライアント発行、SDK 利用。
2. [アーキテクチャ](ARCHITECTURE.md): コンポーネント、認証フロー、データモデル、環境変数。
3. [デプロイと運用](deployment.md): 本番設定、構成生成、デプロイ、バックアップ、監視。
4. [図解](diagrams.md): Mermaid のシーケンス図と ER 図。

## Swagger の更新

API 注釈を変更した場合は次を実行します。

```bash
make swagger
go test ./internal/http -run TestSwaggerDocsAreServed -count=1
```

生成物は `docs/swagger/` に出力されます。
