package domain

type AuthType string

const (
	AuthNone   AuthType = "none"
	AuthBearer AuthType = "bearer"
	AuthBasic  AuthType = "basic"
	AuthAPIKey AuthType = "apikey"
)

type APIKeyLocation string

const (
	APIKeyInHeader APIKeyLocation = "header"
	APIKeyInQuery  APIKeyLocation = "query"
)

type BearerAuth struct {
	Token string // may contain a {{variable}} reference
}

type BasicAuth struct {
	Username string
	Password string // may contain a {{variable}} reference; redacted in reports
}

type APIKeyAuth struct {
	Key      string
	Value    string         // may contain a {{variable}} reference; redacted in reports
	Location APIKeyLocation // where to attach the key
}

type Auth struct {
	Type   AuthType
	Bearer *BearerAuth
	Basic  *BasicAuth
	APIKey *APIKeyAuth
}
