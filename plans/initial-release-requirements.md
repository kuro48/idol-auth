# Initial Release Requirements

## 1. Objective

複数アプリケーションで共通利用できるアカウント認証基盤を構築する。

本システムは Go で実装するバックエンドアプリケーションを中心に構成するが、認証・認可の中核ロジックは自前実装せず、実績のある OSS / ライブラリを利用する。

初期リリースでは、共通アカウント、必須 MFA、OIDC ベースの SSO を成立させることを目的とする。

## 2. System Boundary

### 2.1 This system is responsible for

- 共通アカウントの登録と認証
- ログイン識別子の検証
- MFA の必須化
- OAuth2 / OIDC ベースの認証連携
- 複数アプリ間の SSO
- セッション管理
- OAuth client 管理
- 監査ログ
- 管理者向けの認証運用 API

### 2.2 This system is not responsible for

- 各アプリの業務プロフィール管理
- 氏名、住所、生年月日など広い個人情報の集中管理
- 各アプリ固有の権限モデル
- 業務データ管理

## 3. Core Architecture Policy

- Go アプリは auth facade / admin API / integration layer に責務を限定する
- 認証本体は Ory Kratos と Ory Hydra を第一候補とする
- ブラウザ認証は browser flow を前提とする
- 各アプリは OIDC client として認証基盤に接続する
- 共通基盤は認証専用とし、個人情報の巨大な共通保管庫にしない

## 4. Identity Model

### 4.1 Account policy

- 1 ユーザーは 1 共通アカウントを持つ
- そのアカウントは複数アプリから共通利用できる
- 各アプリは共通アカウントの `sub` を利用して自アプリ内ユーザーと対応づける

### 4.2 Login identifier policy

- ログイン識別子は `メールアドレス` または `電話番号` のどちらか一方を primary として登録する
- 1 アカウントにおいて primary identifier は 1 つのみとする
- 登録時にユーザーが primary identifier を選択する
- primary identifier が未検証の状態では本登録完了とみなさない

### 4.3 Minimal shared account data

共通基盤が保持する情報は最小限に制限する。

- user id
- primary identifier
- identifier verification state
- MFA enrollment state
- account status
- sessions
- authentication events
- audit logs

## 5. Authentication Requirements

### 5.1 Supported login methods

- primary identifier + password
- primary identifier に対する検証フロー

初期リリースでは password を採用するが、将来の passkey / WebAuthn 拡張を妨げない構成にする。

### 5.2 MFA policy

- MFA は初期リリースから必須
- すべてのユーザーは初回登録または初回ログイン完了までに MFA を設定する
- MFA の手段は TOTP のみをサポート対象とする

### 5.3 MFA security policy

- 標準の推奨 MFA は TOTP とする
- 将来的に WebAuthn / passkey を追加可能な設計にする

## 6. SSO Requirements

- 初期リリースでアプリ間 SSO を提供する
- 認証プロトコルは OpenID Connect を採用する
- 各アプリは個別の OAuth / OIDC client として登録する
- 一度認証済みのユーザーが別アプリへ遷移した際、再認証を最小化できること
- SSO 実現のため、共通認証セッションを中央管理する

### 6.1 If SSO were not included

以下の状態になるため、初期リリースでは採用しない。

- アカウントは共通でも各アプリで再ログインが必要
- MFA がアプリごとに繰り返されやすい
- ユーザー体験が悪化する
- 後から SSO を追加する際にセッション設計をやり直す可能性が高い

## 7. OAuth2 / OIDC Requirements

- ブラウザアプリは `Authorization Code Flow + PKCE` を標準とする
- `client_credentials` は machine-to-machine 用途に限定する
- public client は PKCE 必須
- confidential client は secret の安全な保管を前提とする
- client ごとに redirect URI を厳格に管理する
- app ごとに scope を制御する
- トークンに含める claims は最小限に抑える

### 7.1 Minimal claims policy

初期リリースのクライアントに渡す情報は原則以下に限定する。

- `sub`
- 認証済み状態
- 必要最小限の認証文脈

メールアドレス、電話番号、その他個人情報は、明示的な要件がない限りトークンに含めない。

## 8. Privacy and Personal Data Policy

- 個人情報は原則として各アプリが保持する
- 共通認証基盤は認証に必要な最小情報のみ保持する
- あるアプリが不要な個人情報を共通基盤から取得できないようにする
- 個人情報をアプリ間で自動共有しない
- 管理者権限でも無制限閲覧を前提にしない

## 9. Security Requirements

- パスワードハッシュ、セッション、CSRF、OIDC、OAuth2 トークン処理は自前実装しない
- TLS 前提で運用する
- secure cookie を利用する
- rate limiting を導入する
- brute force 対策を入れる
- account enumeration 対策を入れる
- audit log を保存する
- secrets は environment variables または secret manager で管理する
- 管理 API と一般ユーザー向け API の認可境界を分離する

## 10. Admin and Operational Requirements

- 管理者はユーザーの検索、停止、再有効化、セッション失効を行える
- 管理者は OAuth / OIDC client を登録・失効できる
- すべての重要操作に監査証跡を残す
- 障害時に session revoke / client revoke ができる
- バックアップ / リストア手順を整備する

## 11. Initial Release Scope

初期リリースに含める。

- 共通アカウント登録
- primary identifier の選択
- primary identifier の検証
- password 認証
- 必須 MFA
- OIDC ベースの SSO
- 複数アプリ対応 client 管理
- 基本的な管理 API
- 監査ログ

初期リリースに含めない。

- 広範な共通プロフィール管理
- アプリ横断の個人情報統合
- 高度な認可基盤
- ソーシャルログイン
- WebAuthn / passkey 本実装

## 12. Open Decisions Kept for Detailed Design

要件としては固定したが、詳細設計で詰める。

- identifier の重複許可ポリシー
- SMS 配信事業者
- メール配信事業者
- recovery flow の詳細
- global logout の扱い
- app ごとの logout 連携方式
- admin role の粒度

## 13. Recommended Next Design Step

次は以下を設計する。

1. システム構成図
2. ユーザーフロー
3. 認証シーケンス
4. ID schema
5. API 一覧
6. 運用者権限モデル
