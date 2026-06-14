# refactor-instructions.md — idol-auth リファクタリング指示書

> この文書は実装担当モデル(Codex / Opus 等)向けの作業指示書である。
> 目的は「既存仕様を壊さず、技術的負債を減らし、今後変更しやすい状態にする」こと。
> 見た目の綺麗さや全面書き換えは目的ではない。証拠なく削除・書き換えをしないこと。

---

## Objective

1. ベースライン(現在の全テスト緑)を維持したまま、以下を達成する:
   - `backend/internal/http/router.go`(2105行)の責務分割
   - フロントエンド `useSession` のAPI契約バグ修正(テストファースト)
   - 明確に壊れている/死んでいるビルド・CI・Make 設定の整理
   - フロントエンドの lint をCIに追加し、最小限のユニットテスト基盤を導入
2. 仕様判断が必要な項目(SSRページ削除、demoサービス復活/削除、JSON契約変更など)は **実装せず提案に留める**。本書末尾の「Open Questions」が未回答の項目には着手しない。

---

## Project Understanding

### プロダクト概要
**idol-auth** は Ory Kratos(認証/ID管理)+ Ory Hydra(OAuth2/OIDC)を中核にした、マルチアプリ向け共有アカウント基盤(IdP)。アプリ開発者はアプリ登録申請 → 承認 → OIDCクライアント発行 → SDK経由でPKCEログイン、という流れで利用する。UI言語は日本語、「推しメンカラー」テーマ機能を持つ。

### リポジトリ構成
| パス | 役割 |
|---|---|
| `backend/` | Go モジュール `github.com/kuro48/idol-auth`。APIサーバ本体 + 周辺コマンド |
| `backend/cmd/server` | メインAPIサーバ(エントリーポイント、DI組み立て、削除ワーカー起動) |
| `backend/cmd/portal` | Kratos登録プロキシ + Turnstile bot対策の独立サービス(`internal/demo` パッケージを利用) |
| `backend/cmd/migrate` / `configcheck` / `adminctl` | DBマイグレーション / 設定検証 / 管理CLI |
| `backend/internal/http` | ルーター、ハンドラ、ミドルウェア、SSRテンプレート(最大の負債領域) |
| `backend/internal/domain/*` | account / admin / app / appreg / audit / profile ドメインサービス |
| `backend/internal/infra/*` | db(pgx) / hydra / kratos / mail(SMTP) / webhook クライアント |
| `backend/internal/demo` | portal が使う Kratos フロー描画・Turnstile・テーマ処理(名前と実態が乖離) |
| `backend/internal/infra/db/migrations` | golang-migrate 形式 SQL(000001〜000013) |
| `frontend/` | React 19 + Vite + TanStack Router/Query の SPA。本番は Caddy コンテナ + Traefik ルーティング |
| `sdk/` | `@idol-auth/client` — public API 用 TS SDK(PKCEヘルパー含む、CJS/ESM dual build) |
| `docs-site/` | **残骸**(後述 D9)。ドキュメントは SPA `/docs` に移行済み |
| `backend/docker-compose.production.yml` | 本番: Traefik + cloudflared + Kratos/Hydra/app/frontend/portal |
| `docker-compose.yml`(ルート) | 開発用 compose(**現在壊れている — D1 参照**) |

### データフロー(本番)
```
Cloudflare Tunnel → Traefik
  ├─ APP_HOSTNAME + /v1|/login|/register|/uploads|/legal|/account/restore|/healthz|/readyz → app (Go, :8080) [priority=10]
  ├─ APP_HOSTNAME その他すべて → frontend (Caddy SPA fallback) [priority=1]
  ├─ HYDRA_HOSTNAME → hydra :4444
  └─ PORTAL_HOSTNAME → portal :3003
app → Postgres(スキーマ: public/kratos/hydra 分離), Kratos admin/public, Hydra admin/public, SMTP, webhook配送
```

### 検証コマンド(現状)
- `cd backend && go build ./...` / `go test ./...`(CIは `-race` 付き)— **2026-06-12 時点で全パス確認済み**
- `cd backend && go run github.com/securego/gosec/v2/cmd/gosec@latest -exclude-dir=dist -exclude=G124,G203,G301,G306,G703,G704 ./...`(CI同等)
- `cd frontend && npm ci && npm run build`(tsc -b + vite build。CIはこれのみ。lintはCI未実行)
- `make swagger` — Swagger 再生成(`backend/docs/swagger/`)
- `make e2e` — `RUN_E2E=1` で `backend/integration/` を実行。**`APP_URL`(デフォルト `http://localhost:3002` = demoサービス)に依存(D1)**

---

## Behaviors To Preserve(絶対に壊してはいけない既存挙動)

1. **OAuth2/OIDC フロー全体**: `/v1/auth/login|consent|logout` の Hydra チャレンジ処理、consent画面(CSRFトークン+nonce付きCSP)、redirect URL検証(`validateRedirectURL`)。
2. **認証・認可ミドルウェアの判定ロジック**:
   - `adminAuth`: bootstrap token(constant-time比較)→ セッション(AAL2必須 + email/role allowlist)→ 失敗レートリミット。CIDR制限。
   - `accountAuth` / `appTokenAuth` / `accountUIAuth` の判定順序と返すステータスコード。
   - CSRF: `sameOriginBrowserRequest`(ヘッダ無しは**許可** — 非ブラウザAPIクライアント救済)と `sameOriginAdminRequest`(ヘッダ無しは**拒否**)の**意図的な差**。consent専用CSRFクッキー(Path/SameSite=Strict/MaxAge=600)。
3. **公開APIのパス・リクエスト/レスポンス形状**: `/v1/public/*`(SDKが依存)、`/v1/apps/self/*`(外部アプリが管理トークンで依存)、`/v1/admin/*`、`/v1/account/*`、`/v1/auth/session` のJSONフィールド名(snake_case / 一部PascalCase含め**現状のまま**)。
4. **`/v1/auth/session` は未認証でも 200 + `{"authenticated": false}` を返す**(401ではない)。フロント修正(D2)はこの契約に合わせる方向で行い、バックエンドは変えない。
5. **セキュリティヘッダとCSP**(`securityHeaders`、consentページの nonce CSP)、レートリミッタの閾値(authFailure 5/5min、credential 5/min、theme 10/min、global 60/min)。
6. **DBスキーマとmigration履歴**: 既存 migration ファイルの変更・リネーム禁止。
7. **config バリデーション**(production時の必須項目・HTTPS強制・弱トークン拒否・sslmode検査)。
8. **Traefik ルーティングルールと優先度**(`backend/docker-compose.production.yml`)。
9. **graceful shutdown と deletion worker**(`cmd/server/main.go`)。
10. **SDK(`sdk/`)の公開API**(`exports`、PKCEヘルパー)。
11. SSR画面(`/login`, `/register`, `/legal/*`, `/account/restore`)— 本番Traefikで実際に到達するSSRページ。

---

## Non-Negotiables

- 最初に `git status` を確認し、未コミット変更があれば**作業を開始せず報告**する。自分の変更と既存変更を混ぜない。
- 編集前にベースライン検証(Baseline Commands 全部)を実行し結果を記録する。
- 変更は小さく、フェーズごとに独立して revert 可能な単位でコミットする(conventional commits: `refactor:` / `fix:` / `test:` / `ci:` / `chore:`)。
- 無関係な整形・「ついで」リファクタリング禁止。gofmt が変える範囲以外のスタイル変更をしない。
- 既存挙動の変更禁止(D2 のバグ修正を除く。D2 もテストで挙動を固定してから)。
- 正しい仕様が判断できない場合は実装を止めて質問する。
- 各フェーズ完了ごとに Verification Requirements を実行する。
- ルート直下の `.env` / `portal`(25MBバイナリ)/ `dist/` / `*.png` / `docs-site/` は **git未追跡のローカル成果物**。絶対にコミットしない。削除もしない(人間のローカル作業物の可能性があるため)。
- `git push` / PR作成 / force push は指示がない限り行わない。

---

## Stop And Ask Conditions(即時停止して質問する条件)

- Open Questions(Q1〜Q5)に関わるコードに踏み込む必要が生じたとき。
- テストと実装が矛盾しているのを発見したとき(テストを書き換えて通すことを禁止)。
- 公開APIのJSONフィールド名・ステータスコード・パスを変えないと先に進めないとき。
- DB schema、保存済みデータ、migration に影響するとき。
- 認証・CSRF・レートリミット・CSPの**判定結果**が変わる可能性のある変更。
- Kratos/Hydra/SMTP/webhook/Turnstile など外部連携の呼び出し内容が変わるとき。
- 削除候補コードが本当に不要だと**コード上の証拠**(参照ゼロ + ルーティング上到達不能)で示せないとき。

---

## Baseline Commands

作業開始時に以下を実行し、結果(成功/失敗、テスト数)を記録すること:

```bash
git status && git log --oneline -5
cd backend && go build ./... && go test -race ./...
cd backend && go run golang.org/x/vuln/cmd/govulncheck@latest ./...   # 時間がかかる場合はCI結果参照で可
cd frontend && npm ci && npm run build && npm run lint
cd sdk && npm ci && npm test    # tsc build + node --test
```

既知のベースライン(2026-06-12):
- `go build ./...` / `go test ./...` : **全パス**
- frontend `npm run build` : CIでパス(直近コミットでCI追加済み)
- `npm run lint`(frontend)はCI未実行のため、**初回実行で失敗する可能性あり**。失敗してもベースラインとして記録し、Phase 4 で対処。
- `make up`(ルート compose)は **壊れている**(D1)。実行不要。
- `make docs` / `make docs-dev` は **壊れている**(D9)。実行不要。

---

## Debt Map

凡例 — 実装可否: ✅ 今実装してよい / ⚠️ 条件付き(記載の範囲のみ) / ❌ 提案のみ(Open Questions 回答待ち)

### D1. 開発用 docker-compose の `demo` サービスがビルド不能 — ❌(Q1)
- **根拠**: `docker-compose.yml:139-160` が `target: demo`、`backend/Dockerfile:21-23` が `./cmd/demo` をビルドするが、`backend/cmd/` に `demo` ディレクトリが存在しない(`adminctl configcheck migrate portal server` のみ)。backend/frontend 分割(commit `fbdeb62`)時に消えたとみられる。
- **影響**: `make up` が失敗 → ローカル統合環境が起動不能。`make wait` / `verify-local` / `make e2e`(`APP_URL=http://localhost:3002` = demo)も連鎖的に死んでいる。
- **リスク**: demo を「削除」するなら e2e ハーネスの代替が必要。「復活」なら git 履歴から `cmd/demo` を復元する必要がある。どちらが正かはプロダクト判断。
- **改善案**: Q1 の回答に従う。(a) 復活: `git log --all --oneline -- backend/cmd/demo` で削除コミットを特定し復元。(b) 削除: compose の demo サービス、Dockerfile の demo ターゲット、Makefile の DEMO_* 変数・wait・e2e の APP_URL 依存を一括整理し、e2e の代替手順を提案。
- **検証**: `make up && make wait && make check-health`、`make e2e`。

### D2. frontend `useSession` がバックエンドのセッション契約と不一致(実バグ)— ✅(テストファースト必須)
- **根拠**:
  - バックエンド `router.go` `handleSession` は未認証でも **200** で `SessionView`(snake_case: `identity_id`, `email`, `roles`, `authenticated`)を返す。`roles` は `omitempty`。
  - `frontend/src/lib/auth/useSession.ts` は camelCase の `Session { identityId, email, roles }` を期待し、401 のときのみ null とする。
  - 結果: ① `isAuthenticated` は常に true(未認証でも 200 が返るため)。② 未認証時 `roles` が欠落し `data?.roles.includes('admin')` が **TypeError を投げ、AppShell 配下(/account, /developer/*, /admin/*)が未ログイン時にクラッシュし得る**。③ `session.email` 等は常に undefined。
  - 正しい型は既に `frontend/src/lib/api/types.ts` の `SessionView` として存在する(未使用)。
- **影響**: AppShell の全画面のログイン状態表示・admin ナビ表示。
- **リスク**: 低。バックエンド契約はテストで固定済み(`auth_test.go` 等)。フロントのみ修正。
- **改善案**: ① vitest を導入(D12 と同時)。② `useSession` に対し「`{authenticated:false}` → 未認証扱い」「`{authenticated:true, roles:['admin'], ...}` → isAdmin=true」のテストを先に書き(RED)、③ `types.ts` の `SessionView` を使う実装に修正(GREEN)。`roles` 欠落時に落ちないこと(`?? []`)。バックエンドは一切変更しない。
- **検証**: 新規ユニットテスト + `npm run build`。可能なら `make frontend-dev` + backend起動でブラウザ確認(未ログインで `/account` を開いてクラッシュしないこと)。

### D3. AppShell のナビが存在しないルートを指す — ⚠️
- **根拠**: `frontend/src/components/layout/AppShell.tsx` の `buildSections` に `/account/profile`, `/account/sessions` があるが、`frontend/src/routeTree.ts` に該当ルートはない → クリックで NotFoundPage。
- **影響**: UXのみ。API影響なし。
- **改善案(許可範囲)**: SPA移行が未完了であることを示す痕跡。**ページを新規実装してはならない**(スコープ外)。リンクを消すか残すかは Q2 の回答に従う。回答がなければ現状維持。
- **検証**: ブラウザ目視。

### D4. バックエンドSSR画面とSPAの二重実装 / 本番で到達不能なSSR — ❌(Q3)
- **根拠**: backend は `/account`(account_center, テンプレ988行)、`/settings`(+ `/v1/settings/flow`)、`/developer/app-requests/*`(HTML版)、`/admin-ui/*` のSSR画面を持つ。一方、本番 Traefik(`backend/docker-compose.production.yml` の app ルーター rule)は `/v1, /login, /register, /uploads, /legal, /account/restore, /healthz, /readyz` のみを backend に向け、**それ以外はすべて SPA が受ける**。つまり本番では:
  - `/account` → SPA(AccountOverviewPage)。SSR account_center は到達不能。
  - `/developer/app-requests` → SPA。SSR版は到達不能。
  - `/settings` → SPA に該当ルートが**ない** → NotFoundPage(SSR settings 画面は到達不能。ただし API `/v1/settings/flow` は到達可能)。
  - `/admin-ui/*` → SPA に該当ルートがない → NotFoundPage。
  - 例外: portal は `PORTAL_ACCOUNT_CENTER_URL: ${APP_BASE_URL}/account/` を参照しており、これはSPA側 `/account` に落ちる(動作はする)。
- **なぜ負債か**: 同一機能のUIが2系統(Go template ~3,000行 + React)あり、修正が二重化する。SSR側はテストも厚く(admin_test.go 1480行等)、削除判断を誤ると本物の機能(/settings のSPA未実装分)を失う。
- **リスク**: 高。`/settings` と `/admin-ui` は SPA 未移植のため、SSR削除=機能喪失。
- **改善案**: **提案のみ**。Q3 の回答(SPAが正、SSRは段階的廃止/SSR維持/未定)を得てから、削除対象とSPA移植の不足分を列挙した移行計画を別途出す。今回のリファクタでは SSR ハンドラ・テンプレート・テストを**一切削除しない**。
- **検証**: N/A(提案のみ)。

### D5. `backend/internal/http/router.go` が2105行で責務過積載 — ✅
- **根拠**: ルート定義(196-427行)、約30個のHTTPハンドラ、認証/CSRF/CORS/securityHeadersミドルウェア、エラーマッパ(`writeAuthError`/`writeDomainError`/`writeAccountError`)、JSONヘルパ、**約300行のインラインconsent HTMLテンプレート**(1620-1921行)が同居。コーディング規約の800行上限の2.6倍。
- **影響**: 変更時の認知負荷・コンフリクト頻発。テストは `router_test.go` ほか同パッケージに多数あり、安全網は厚い。
- **リスク**: 低〜中。**同一パッケージ内のファイル分割のみ**なら識別子・挙動は不変。
- **改善案**(パッケージは `http` のまま、移動のみ。export 追加・シグネチャ変更・ロジック変更禁止):
  - `router.go` — `RouterConfig`、`NewRouter`(ルート定義)、`server` 構造体のみ残す(目標 <500行)
  - `middleware.go` — adminAuth / accountAuth / appTokenAuth / CSRF系 / corsMiddleware / securityHeaders / sameOrigin系 / adminIPAllowed / requestIsSecure
  - `handlers_auth.go` — session / theme / login / consent / logout 系ハンドラ
  - `handlers_admin.go` — /v1/admin 系ハンドラ + resolveUserRef
  - `handlers_account.go` — /v1/account, /v1/apps/self 系ハンドラ
  - `respond.go` — writeJSON / writeError / decodeJSON / エラーマッパ / wantsJSON
  - `consent_page.go` — writeConsentPage とテンプレート(既存 `*_templates.go` の慣例に合わせる)
  - 分割は関数単位のカット&ペーストで行い、**1関数も書き換えない**こと。
- **検証**: `go build ./... && go test -race ./...` が無修正で全パス。`make swagger` を実行し `backend/docs/swagger/swagger.json` に差分がないこと(差分が出たら移動ミス)。gosec も再実行。

### D6. `NewRouter` の可変長引数ハック — ✅
- **根拠**: `router.go:196` `NewRouter(cfg, adminSvc, readiness, authSvc, accountSvcs ...AccountService)` — 後方互換のための variadic。他の任意サービスは `RouterConfig` のフィールドで渡している(`ProfileSvc` 等)非対称な設計。
- **影響**: 呼び出し側は `cmd/server/main.go` とテストのみ(リポジトリ内検索で確認すること)。外部公開APIではない。
- **改善案**: `RouterConfig` に `AccountSvc AccountService` を追加し、variadic を廃止。全呼び出し箇所(テスト含む)を機械的に更新。
- **リスク**: 低。コンパイルエラーで漏れが検出できる。
- **検証**: `go build ./... && go test -race ./...`。

### D7. `sameOriginBrowserRequest` / `sameOriginAdminRequest` の重複 — ✅(慎重に)
- **根拠**: `router.go:1387-1422`。2関数はヘッダ無し時のフォールバック(browser=true / admin=false)だけが違う。この差は**意図的なセキュリティ判断**(コメントに明記)。
- **改善案**: `sameOriginRequest(r *http.Request, allowMissingHeaders bool) bool` に統合し、既存2関数は薄いラッパとして残すか呼び出し側で直接使う。**真理値表が1ケースも変わらないこと**をテーブル駆動テストで先に固定してから統合する(現状この2関数の直接テストが無ければ追加)。
- **検証**: 新規テーブル駆動テスト + 既存テスト全パス。

### D8. JSON契約の不統一(PascalCaseが漏れている)— ❌(Q4)
- **根拠**: `frontend/src/lib/api/types.ts` のコメントが示す通り、`appreg`(AppRequest)と `audit`(AuditLog)の構造体に JSON タグがなく、Goフィールド名(`ID`, `IdentityID`, `OccurredAt` …)がそのまま `/v1/developer/app-requests` / `/v1/admin/audit-logs` 等のレスポンスに露出。他は snake_case。
- **なぜ負債か**: API契約としての一貫性欠如。SPA・外部利用者がPascalCaseに依存し始めている。
- **リスク**: **高 — 公開APIの破壊的変更**。snake_case 化するとSPA(types.ts)と外部クライアントが同時に壊れる。
- **改善案**: 提案のみ。Q4 の回答後、(a) 互換期間を設けたデュアル出力、(b) v2 エンドポイント、(c) 現状維持を比較した提案書を出す。**今回は一切変更しない**。

### D9. docs パイプラインの残骸 — ⚠️(下記範囲のみ実装可)
- **根拠**:
  - `.github/workflows/docs.yml` — 自身のコメントに「This workflow is intentionally disabled — remove the file to clean up.」と書かれた no-op。
  - `Makefile` の `docs` / `docs-dev` ターゲット — `backend/internal/docsfs/dist/`(**存在しないディレクトリ**)へコピーし、`docs-site/`(package.json すら無い残骸、git未追跡)で `npm ci` する → 実行すると必ず失敗。
  - docs の実体は SPA(`frontend/src/routes/docs/`)に移行済み(commit `23dea9d`)。
- **実装可の範囲**: ① `docs.yml` を削除(ファイル自身が削除を指示)。② Makefile から `docs` / `docs-dev` ターゲットを削除(壊れており、参照する `docs-site`/`docsfs` が実在しない)。③ ローカルの `docs-site/` ディレクトリ自体は**触らない**(git未追跡。人間に削除を委ねる旨を報告に書く)。
- **検証**: `make swagger` は引き続き動くこと。`grep -rn "docsfs\|docs-site" Makefile .github backend/` で残参照ゼロ。

### D10. OpenAPI スペックのコピー運用がドリフトする — ✅(小)
- **根拠**: SPA の API リファレンス(`frontend/src/routes/docs/DocsApiPage.tsx`)は `/openapi.json` = `frontend/public/openapi.json` を読むが、これを `backend/docs/swagger/swagger.json` から更新する Make ターゲットが無い(旧 `docs` ターゲットは docs-site にしかコピーしない)。
- **改善案**: Makefile に追加:
  ```make
  frontend-openapi: swagger
  	cp backend/docs/swagger/swagger.json frontend/public/openapi.json
  ```
  併せて現時点の `swagger.json` と `frontend/public/openapi.json` の diff を確認し、差分があれば**報告のみ**(同期コミットするかは人間判断 — APIドキュメントの見た目が変わるため)。
- **検証**: `make frontend-openapi` 成功。

### D11. 嘘になったコメント — ✅(微小)
- **根拠**: `router.go:228-230`「/docs is now served by the SPA frontend (proxied via **nginx**)」— nginx は commit `872bdb1` で Caddy + Traefik に置換済み。また直後のコメント「The backend redirects legacy /docs/* URLs…」に対応する実装は `r.Get("/", …/login redirect)` のみで `/docs` リダイレクトは存在しない(本番では Traefik が /docs をSPAに送るので実害なし)。
- **改善案**: コメントを実態(Traefik が SPA にルーティング)に合わせて修正。**リダイレクト実装の追加はしない**(挙動変更になるため)。
- **検証**: ビルドのみ。

### D12. フロントエンドのテストゼロ + lint がCI外 — ✅
- **根拠**: `frontend/` にテストファイル・テストランナーが一切ない。`package.json` に `lint` スクリプトはあるが `.github/workflows/ci.yml` の frontend ジョブは install + build のみ。
- **改善案**: ① vitest(+ @testing-library/react は必要になるまで入れない)を devDependencies に追加し、`test` スクリプトを定義。② D2 の `useSession` テストと `lib/api/client.ts`(ApiError、204処理)のユニットテストを書く。③ CI の frontend ジョブに `npm run lint` と `npm run test` を追加。lint が既存コードで失敗する場合、**自動修正で挙動が変わらない範囲(import順等)のみ修正**し、ルール無効化や大規模修正が必要なら停止して報告。
- **リスク**: 低。プロダクションコードへの変更は D2 のみ。
- **検証**: `npm run test` / `npm run lint` / `npm run build` 全パス、CI定義のYAML構文確認。

### D13. インメモリ・レートリミッタの単一インスタンス前提 — 記録のみ
- **根拠**: `NewInMemoryRateLimiter`(router.go / ratelimit.go)。水平スケール時にIP別制限が効かなくなる。現構成(単一app)では問題なし。
- **対応**: 変更不要。報告書に「将来 app を複数レプリカにする場合は Redis 等への移行が必要」と記載するのみ。

### D14. `internal/demo` パッケージの名前と実態の乖離 — ❌(提案のみ)
- **根拠**: `internal/demo` は現在 **portal**(本番サービス)の中核実装(Kratosフロー描画1029行、Turnstile、テーマ)。"demo" という名前が「消してよいコード」という誤解を招く(実際 D1 で cmd/demo だけが消えた)。
- **改善案**: `internal/portal` 等へのリネームを提案。ただし Q1(demo復活の可能性)と絡むため、Q1 回答後に判断。今回は触らない。

### D15. ルート直下のローカル成果物 — 対応不要(報告のみ)
- **根拠**: `./portal`(25MB Goバイナリ)、`./dist/`、`./.env`、`./*.png`、`./docs-site/`。すべて .gitignore 済み・未追跡。
- **対応**: コミットに混入させないこと以外、何もしない。報告書で人間にローカル掃除を提案。

---

## Implementation Phases

> 各フェーズは独立コミット。フェーズ末に Verification Requirements を実行。失敗したらそのフェーズ内で解決するか revert して報告。

### Phase 0 — 現状確認(変更なし)
1. `git status`(クリーンでなければ停止・報告)
2. Baseline Commands を全実行し、結果を記録(これが以後の比較基準)
3. frontend `npm run lint` の初回結果を記録(失敗してもこの段階では直さない)

### Phase 1 — 安全網の追加(プロダクションコード変更なし)
1. vitest 導入(`frontend`): devDependency 追加、`"test": "vitest run"` スクリプト、最小設定
2. `useSession` の**現行バグを暴露する**テストを作成(RED を確認)
3. `lib/api/client.ts` のユニットテスト(正常/エラー/204)
4. backend: `sameOriginBrowserRequest` / `sameOriginAdminRequest` の真理値表テストが既存テストに無ければ追加(現行実装の挙動をそのまま固定。GREEN であること)
5. コミット: `test: add frontend unit test harness and same-origin truth-table tests`

### Phase 2 — 明らかに安全な整理
1. D9: `docs.yml` 削除、Makefile `docs`/`docs-dev` ターゲット削除
2. D10: `frontend-openapi` ターゲット追加 + スペック差分の確認(差分は報告のみ)
3. D11: router.go の stale コメント修正
4. コミット: `chore: remove dead docs pipeline, fix stale comments, add openapi sync target`

### Phase 3 — バグ修正(D2)
1. `useSession` を `SessionView`(snake_case + `authenticated` フラグ)準拠に修正、Phase 1 のテストが GREEN になること
2. `roles` 欠落時の安全化(`?? []`)
3. 可能ならローカルでブラウザ確認(未ログインで `/account` がクラッシュしない、ログイン後に email / admin ナビが出る)
4. コミット: `fix(frontend): align useSession with /v1/auth/session contract`

### Phase 4 — CI 強化(D12 残り)
1. CI frontend ジョブに `npm run lint` と `npm run test` を追加
2. lint 失敗が残る場合: 挙動不変の自動修正のみ適用。判断が要るものは停止・報告
3. コミット: `ci(frontend): run lint and unit tests`

### Phase 5 — backend/internal/http の構造整理(D5, D6, D7)
> 1ステップ = 1コミット。各ステップ後に `go build && go test -race ./...` + `make swagger` 差分ゼロ確認。
1. `respond.go` 切り出し(ヘルパ・エラーマッパ)
2. `consent_page.go` 切り出し(consentテンプレート)
3. `middleware.go` 切り出し
4. `handlers_admin.go` / `handlers_account.go` / `handlers_auth.go` 切り出し
5. D7: sameOrigin 統合(Phase 1 のテーブルテストが無修正で通ること)
6. D6: `NewRouter` variadic 廃止 → `RouterConfig.AccountSvc`
7. コミット例: `refactor(http): extract response helpers from router.go`(各ステップ)

### Phase 6 — 提案書の作成(実装しない)
Open Questions(Q1, Q3, Q4)それぞれについて、選択肢・影響範囲・推奨案・移行手順をまとめ、最終報告に含める。**コードは変更しない。**

---

## Verification Requirements

各フェーズ末で必ず:
```bash
cd backend && go build ./... && go test -race ./...
cd frontend && npm run build
```
Phase 1 以降は追加で `cd frontend && npm run test`。
Phase 5 では各ステップ後に `make swagger` を実行し `git diff backend/docs/swagger/` が空であることを確認(空でない=ハンドラ移動時にswaggerコメントを壊した可能性 → 修正)。
Phase 5 完了時に CI 同等の gosec を1回実行:
```bash
cd backend && go run github.com/securego/gosec/v2/cmd/gosec@latest -exclude-dir=dist -exclude=G124,G203,G301,G306,G703,G704 ./...
```
失敗した検証を放置して次フェーズへ進むことを禁止する。

---

## Reporting Format

最終報告には以下を含めること:
1. **Baseline**: Phase 0 で記録したコマンドと結果(そのまま貼る)
2. **フェーズごとの結果表**: コミットハッシュ / 変更ファイル数 / 実行した検証コマンドと結果
3. **最後に実行したコマンドとその出力(要約可、成否は明記)**
4. **Deviation log**: 指示書から逸脱した点・できなかった点・新たに発見した問題
5. **Proposals**: Phase 6 の提案書(Q1, Q3, Q4)
6. **人間へのTODO**: ローカル残骸(D15)の掃除提案、未回答の Open Questions

---

## Out-of-scope Items(今回やらないこと)

- バックエンドSSR画面・テンプレート・関連テストの削除/移植(D4 — Q3待ち)
- demo サービスの復活または削除(D1 — Q1待ち)
- appreg/audit の JSON タグ追加などAPI契約変更(D8 — Q4待ち)
- `internal/demo` パッケージのリネーム(D14)
- SPA への新規ページ実装(/settings, /admin-ui 相当, /account/profile, /account/sessions)
- レートリミッタの分散対応(D13)
- DB schema / migration の変更
- 依存ライブラリのバージョン更新(脆弱性対応が必要な場合のみ停止して報告)
- Kratos / Hydra / Traefik / Caddy 設定の変更
- ルート直下の未追跡ファイルの削除
- git push、PR作成、deploy ブランチ操作

---

## Open Questions(実装前に確認すべき質問)

**Q1. `demo` サービスの扱い(D1)** — `backend/cmd/demo` が存在せず `make up`(開発compose)と `make e2e` が壊れています。(a) git履歴から `cmd/demo` を復元する、(b) demo を開発composeから削除し e2e ハーネスを別の方法(portal や SDKベースのテストアプリ)に置き換える、どちらが正ですか?

**Q2. AppShell の `/account/profile` `/account/sessions` リンク(D3)** — ルート未実装で NotFound に落ちます。リンクを一旦削除しますか、近日実装予定として残しますか?(対応するAPI `/v1/account/profile`, `/v1/account/sessions` はバックエンドに実装済み)

**Q3. バックエンドSSR画面の今後(D4)** — 本番では `/account`・`/settings`・`/developer/app-requests`(HTML)・`/admin-ui` のSSR画面に到達できず、SPAが受けています(`/settings` と `/admin-ui` はSPA未実装のためNotFound)。SPAを正として SSR を段階廃止する方針で確定ですか? `/settings` のSPA移植は予定されていますか?

**Q4. AppRequest / AuditLog のPascalCase JSON(D8)** — `/v1/developer/app-requests` と `/v1/admin/audit-logs` のレスポンスがGoフィールド名のまま露出しています。SDK外の外部クライアントは存在しますか? snake_case への統一(破壊的変更)を計画してよいですか?

**Q5. frontend lint の現状** — CIに lint を追加した際、既存コードでエラーが出た場合、ルール調整(緩和)とコード修正のどちらを優先しますか?(デフォルト: 挙動不変の修正のみ行い、判断が要るものは報告)
