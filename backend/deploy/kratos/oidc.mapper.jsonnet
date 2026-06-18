// Maps OIDC provider claims to Kratos identity traits.
// Used by Google, X (Twitter), and Apple OIDC providers.
local claims = std.extVar('claims');
{
  identity: {
    traits: {
      [if "email" in claims then "email" else null]: claims.email,
      primary_identifier_type: "email",
    },
  },
}
