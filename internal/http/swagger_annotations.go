package http

// @Summary Liveness check
// @Description プロセスの liveness を返します。
// @Tags Health
// @Produce json
// @Success 200 {object} swaggerStatusResponse
// @Router /healthz [get]
func swaggerHealthzDoc() {}

// @Summary Readiness check
// @Description 依存サービスの readiness を返します。未準備の場合は 503 を返します。
// @Tags Health
// @Produce json
// @Success 200 {object} swaggerStatusResponse
// @Failure 503 {object} swaggerStatusResponse
// @Router /readyz [get]
func swaggerReadyzDoc() {}

// @Summary Browser flow provider URLs
// @Description Kratos / Hydra の browser flow URL 一覧を返します。
// @Tags Auth
// @Produce json
// @Success 200 {object} ProviderView
// @Router /v1/auth/providers [get]
func swaggerAuthProvidersDoc() {}

// @Summary Current session
// @Description 現在の Kratos セッション要約を返します。未認証でも 200 で `authenticated=false` を返します。
// @Tags Auth
// @Produce json
// @Success 200 {object} SessionView
// @Failure 503 {object} swaggerErrorResponse
// @Router /v1/auth/session [get]
func swaggerAuthSessionDoc() {}

// @Summary Update theme preference
// @Description 推しメンカラー設定を更新します。Kratos セッション Cookie が必要です。
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body swaggerThemePreferenceRequest true "theme preference"
// @Success 200 {object} map[string]string
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 503 {object} swaggerErrorResponse
// @Router /v1/auth/theme [post]
func swaggerAuthThemeDoc() {}

// @Summary Logout start
// @Description Accept ヘッダーに応じて logout URL の JSON 返却またはブラウザリダイレクトを返します。
// @Tags Auth
// @Produce json
// @Success 200 {object} swaggerLogoutStartResponse
// @Success 303 {string} string "See Other"
// @Router /v1/auth/logout [post]
func swaggerAuthLogoutStartDoc() {}

// @Summary Logout entry
// @Description `logout_challenge` がなければ logout URL を返すか Hydra へリダイレクトし、あれば Hydra の logout callback を処理します。
// @Tags Auth
// @Produce json
// @Param logout_challenge query string false "Hydra logout challenge"
// @Success 200 {object} swaggerLogoutStartResponse
// @Success 302 {string} string "Found"
// @Success 303 {string} string "See Other"
// @Failure 400 {object} swaggerErrorResponse
// @Failure 502 {object} swaggerErrorResponse
// @Failure 503 {object} swaggerErrorResponse
// @Router /v1/auth/logout [get]
func swaggerAuthLogoutDoc() {}

// @Summary Hydra login challenge
// @Description Hydra の login_challenge を処理し、Kratos login/settings か Hydra accept へリダイレクトします。
// @Tags Auth
// @Produce html
// @Param login_challenge query string true "Hydra login challenge"
// @Success 302 {string} string "Found"
// @Failure 400 {object} swaggerErrorResponse
// @Failure 502 {object} swaggerErrorResponse
// @Failure 503 {object} swaggerErrorResponse
// @Router /v1/auth/login [get]
func swaggerAuthLoginDoc() {}

// @Summary Hydra consent challenge
// @Description first-party や skip consent の場合は自動 accept、確認が必要な場合は HTML を返します。
// @Tags Auth
// @Produce html
// @Param consent_challenge query string true "Hydra consent challenge"
// @Success 200 {string} string "HTML consent page"
// @Success 302 {string} string "Found"
// @Failure 400 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Failure 502 {object} swaggerErrorResponse
// @Failure 503 {object} swaggerErrorResponse
// @Router /v1/auth/consent [get]
func swaggerAuthConsentDoc() {}

// @Summary Submit consent decision
// @Description HTML consent 画面の submit 先です。CSRF Cookie と csrf_token の一致が必要です。
// @Tags Auth
// @Accept application/x-www-form-urlencoded
// @Produce html
// @Param consent_challenge formData string true "Hydra consent challenge"
// @Param csrf_token formData string true "CSRF token"
// @Param action formData string true "accept or deny" Enums(accept,deny)
// @Success 302 {string} string "Found"
// @Failure 400 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Failure 502 {object} swaggerErrorResponse
// @Failure 503 {object} swaggerErrorResponse
// @Router /v1/auth/consent [post]
func swaggerAuthConsentSubmitDoc() {}

// @Summary List apps
// @Description 登録済みアプリ一覧を返します。
// @Tags Admin Apps
// @Produce json
// @Security BearerAuth
// @Success 200 {object} swaggerAppListResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Failure 503 {object} swaggerErrorResponse
// @Router /v1/admin/apps [get]
func swaggerAdminListAppsDoc() {}

// @Summary Create app
// @Description app を作成します。`client` ブロックまたは `redirect_uris` を指定すると OIDC client も同時作成します。
// @Tags Admin Apps
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body swaggerCreateAppRequest true "app creation request"
// @Success 201 {object} swaggerCreateAppResponse
// @Success 201 {object} swaggerCreateAppWithClientResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Failure 409 {object} swaggerErrorResponse
// @Failure 500 {object} swaggerErrorResponse
// @Router /v1/admin/apps [post]
func swaggerAdminCreateAppDoc() {}

// @Summary Issue or rotate app management token
// @Description app-scoped user management API 用の management token を発行します。既存 token は無効化されます。
// @Tags Admin Apps
// @Produce json
// @Security BearerAuth
// @Param appID path string true "App ID"
// @Success 200 {object} swaggerManagementTokenResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Failure 404 {object} swaggerErrorResponse
// @Router /v1/admin/apps/{appID}/management-token [post]
func swaggerAdminIssueManagementTokenDoc() {}

// @Summary List OIDC clients
// @Description 指定 app に紐づく OIDC client 一覧を返します。
// @Tags Admin Apps
// @Produce json
// @Security BearerAuth
// @Param appID path string true "App ID"
// @Success 200 {object} swaggerOIDCClientListResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Router /v1/admin/apps/{appID}/clients [get]
func swaggerAdminListOIDCClientsDoc() {}

// @Summary Create OIDC client
// @Description 指定 app 向けに OIDC client を作成します。
// @Tags Admin Apps
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param appID path string true "App ID"
// @Param request body swaggerCreateOIDCClientRequest true "OIDC client creation request"
// @Success 201 {object} swaggerCreateOIDCClientResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Failure 404 {object} swaggerErrorResponse
// @Failure 409 {object} swaggerErrorResponse
// @Router /v1/admin/apps/{appID}/clients [post]
func swaggerAdminCreateOIDCClientDoc() {}

// @Summary Search users
// @Description Kratos identity を identifier と state で検索します。
// @Tags Admin Users
// @Produce json
// @Security BearerAuth
// @Param identifier query string false "email / phone partial match"
// @Param state query string false "identity state" Enums(active,inactive)
// @Success 200 {object} swaggerIdentityListResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Failure 502 {object} swaggerErrorResponse
// @Router /v1/admin/users [get]
func swaggerAdminSearchUsersDoc() {}

// @Summary Patch user state or roles
// @Description `state` と `roles` のどちらか、または両方を更新します。
// @Tags Admin Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userRef path string true "Identity UUID or URL-encoded email"
// @Param request body swaggerPatchUserRequest true "patch user request"
// @Success 200 {object} swaggerIdentity
// @Success 200 {object} swaggerRolesUpdateResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Failure 404 {object} swaggerErrorResponse
// @Failure 502 {object} swaggerErrorResponse
// @Router /v1/admin/users/{userRef} [patch]
func swaggerAdminPatchUserDoc() {}

// @Summary Revoke user sessions
// @Description identity に紐づくアクティブセッションを失効します。
// @Tags Admin Users
// @Produce json
// @Security BearerAuth
// @Param userRef path string true "Identity UUID or URL-encoded email"
// @Success 204 {string} string "No Content"
// @Failure 401 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Failure 404 {object} swaggerErrorResponse
// @Failure 502 {object} swaggerErrorResponse
// @Router /v1/admin/users/{userRef}/revoke-sessions [post]
func swaggerAdminRevokeUserSessionsDoc() {}

// @Summary Delete user
// @Description identity を削除します。
// @Tags Admin Users
// @Produce json
// @Security BearerAuth
// @Param userRef path string true "Identity UUID or URL-encoded email"
// @Success 204 {string} string "No Content"
// @Failure 401 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Failure 404 {object} swaggerErrorResponse
// @Failure 502 {object} swaggerErrorResponse
// @Router /v1/admin/users/{userRef} [delete]
func swaggerAdminDeleteUserDoc() {}

// @Summary List audit logs
// @Description actor / target / event type と pagination で監査ログを取得します。
// @Tags Admin Audit
// @Produce json
// @Security BearerAuth
// @Param actor_type query string false "actor type"
// @Param actor_id query string false "actor id"
// @Param target_type query string false "target type"
// @Param target_id query string false "target id"
// @Param event_type query string false "event type"
// @Param limit query int false "limit"
// @Param offset query int false "offset"
// @Success 200 {object} swaggerAuditLogListResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Failure 502 {object} swaggerErrorResponse
// @Router /v1/admin/audit-logs [get]
func swaggerAdminAuditLogsDoc() {}

// @Summary Current shared account
// @Description 認証中の共有アカウント本体と、この identity が接続している app membership 一覧を返します。
// @Tags Account
// @Produce json
// @Security BearerAuth
// @Success 200 {object} swaggerAccountOverviewResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 503 {object} swaggerErrorResponse
// @Router /v1/account [get]
func swaggerAccountOverviewDoc() {}

// @Summary Disconnect app from current account
// @Description 現在の共有アカウントから指定 app との membership を解除します。identity 本体は削除しません。
// @Tags Account
// @Produce json
// @Security BearerAuth
// @Param appID path string true "App ID"
// @Success 204 {string} string "No Content"
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 404 {object} swaggerErrorResponse
// @Router /v1/account/apps/{appID} [delete]
func swaggerDisconnectAccountAppDoc() {}

// @Summary Get account deletion request
// @Description 現在の shared account に対する削除予約状態を返します。
// @Tags Account
// @Produce json
// @Security BearerAuth
// @Success 200 {object} swaggerDeletionRequestEnvelope
// @Failure 401 {object} swaggerErrorResponse
// @Router /v1/account/deletion [get]
func swaggerGetDeletionRequestDoc() {}

// @Summary Schedule shared account deletion
// @Description 現在の shared account の完全削除を予約します。app membership だけでなく identity 本体が対象です。
// @Tags Account
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body swaggerScheduleDeletionRequest true "deletion request"
// @Success 202 {object} swaggerDeletionRequestEnvelope
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Router /v1/account/deletion [post]
func swaggerScheduleDeletionDoc() {}

// @Summary Cancel shared account deletion
// @Description 予約済み shared account deletion を取り消します。
// @Tags Account
// @Produce json
// @Security BearerAuth
// @Success 204 {string} string "No Content"
// @Failure 401 {object} swaggerErrorResponse
// @Failure 404 {object} swaggerErrorResponse
// @Router /v1/account/deletion [delete]
func swaggerCancelDeletionDoc() {}

// @Summary List app-scoped users
// @Description app management token に紐づく app から見える membership 一覧を返します。
// @Tags App Self
// @Produce json
// @Security BearerAuth
// @Success 200 {object} swaggerMembershipListResponse
// @Failure 401 {object} swaggerErrorResponse
// @Router /v1/apps/self/users [get]
func swaggerAppSelfListUsersDoc() {}

// @Summary Revoke app-scoped user
// @Description app management token に紐づく app から指定 identity の membership を無効化します。shared account 本体は削除しません。
// @Tags App Self
// @Produce json
// @Security BearerAuth
// @Param identityID path string true "Identity ID"
// @Success 204 {string} string "No Content"
// @Failure 401 {object} swaggerErrorResponse
// @Failure 404 {object} swaggerErrorResponse
// @Router /v1/apps/self/users/{identityID} [delete]
func swaggerAppSelfRevokeUserDoc() {}
