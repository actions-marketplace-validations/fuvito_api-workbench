package redact_test

import (
	"testing"

	"github.com/fuvito/api-workbench/packages/core/redact"
)

func TestCredentialsMasksOnlyThePassword(t *testing.T) {
	for in, want := range map[string]string{
		"postgres://app:s3cretPw@db.internal:5432/orders":      "postgres://app:***@db.internal:5432/orders",
		"cannot reach amqp://app:s3cretPw@mq:5672/ (timeout)":  "cannot reach amqp://app:***@mq:5672/ (timeout)",
		"mongodb+srv://admin:p%40ss@cluster0.example.net/db":   "mongodb+srv://admin:***@cluster0.example.net/db",
		"redis://:s3cretPw@cache:6379":                         "redis://:***@cache:6379",
		"https://user:pa@ss@example.com/x":                     "https://user:***@example.com/x",
		`{"u":"http://a:b1@h.example/p","n":1}`:                `{"u":"http://a:***@h.example/p","n":1}`,
		"two: ftp://a:one@h1/ and ftp://b:two@h2/":             "two: ftp://a:***@h1/ and ftp://b:***@h2/",
		"Server=db;User Id=app;Password=s3cretPw;":             "Server=db;User Id=app;Password=***;",
		"Driver={SQL};Uid=app;Pwd=s3cretPw;":                   "Driver={SQL};Uid=app;Pwd=***;",
		"Uid=app;Pwd={s3cretPw};Database=x":                    "Uid=app;Pwd=***;Database=x",
		"Uid=app;Pwd={s3 cret;Pw};Database=x":                  "Uid=app;Pwd=***;Database=x",
		`Password="s3 cret;Pw";Server=db`:                      `Password="***";Server=db`,
		"password='s3cretPw'":                                  "password='***'",
		"https://api.example.com/i?user=a&password=s3cret&p=2": "https://api.example.com/i?user=a&password=***&p=2",
		"?access_token=abc123&client_secret=def456&api_key=g7": "?access_token=***&client_secret=***&api_key=***",
		"DB_PASSWORD=s3cretPw":                                 "DB_PASSWORD=***",
		"AccountName=x;AccountKey=abc/def+g==;Suffix=core":     "AccountName=x;AccountKey=***;Suffix=core",
		"login failed for password=s3cretPw, retry later.":     "login failed for password=***, retry later.",
		"(token=abc123)":                                       "(token=***)",
		`expected "x"; got "Server=db;Password=s3cretPw"`:      `expected "x"; got "Server=db;Password=***"`,
	} {
		got := redact.Credentials(in)
		if got != want {
			t.Errorf("%q\n  got  %q\n  want %q", in, got, want)
		}
		// baselines are compared by masking both sides, so a second pass must change nothing
		if again := redact.Credentials(got); again != got {
			t.Errorf("not stable: %q became %q", got, again)
		}
	}
}

func TestCredentialsLeavesOrdinaryTextAlone(t *testing.T) {
	for _, in := range []string{
		// addresses without a password
		"https://example.com/help?page=2&sort=name",
		"http://localhost:8080/users/42",
		"http://127.0.0.1:8080",
		"https://[::1]:8443/path",
		"https://git@github.com/org/repo.git",
		"ssh://deploy@host.example.com:22/srv",
		"http://localhost:8080/send?to=a@example.com",
		"http://example.com:80/users/@me",
		"https://example.com/a:b@c",
		`{"url":"http://h.example:80","email":"a@example.com"}`,
		"postgres://app:{{dbPassword}}@db:5432/x",
		"postgres://app:${DB_PASSWORD}@db:5432/x",
		"postgres://app:***@db:5432/x",
		// e-mail addresses
		"ops@example.com",
		"Contact: jane.doe+api@example.co.uk, or mailto:help@example.com",
		// prose, expressions and settings that only mention these words
		"password is required",
		"The password: must have 8 characters",
		"token expired at 12:30; ask the admin@example.com",
		"if password == confirm then ok",
		`password="" is not allowed`,
		"reset_password=true&remember=false",
		"Pwd={{password}};Uid={{user}}",
		"password=$DB_PASSWORD",
		"password=<your password>",
		"PWD=/home/app OLDPWD=/tmp",
		"pageToken=abc123&next_token=def456&token_type=Bearer",
		"session=abc&key=name&auth=none&monkey=1",
		"a = b, x=1, status=ok, secretary=jane, passport=X123",
		"Server=db;Database=orders;User Id=app;",
		"",
	} {
		if got := redact.Credentials(in); got != in {
			t.Errorf("changed ordinary text:\n  %q\n  %q", in, got)
		}
	}
}

func TestAnAddressWithAPasswordKeepsItsHost(t *testing.T) {
	const in = "postgres://app:s3cretPw@db.internal.example.com:5432/orders"
	const want = "postgres://app:***@db.internal.example.com:5432/orders"
	if redact.LooksSecret(in) {
		t.Error("an address is not hidden as a whole")
	}
	if got := redact.Tokens(in); got != want {
		t.Errorf("Tokens: %q", got)
	}
	// the excerpt's e-mail rule used to take "password@host" for an address and hide the host
	if got := redact.Excerpt("upstream "+in+" refused", 200); got != "upstream "+want+" refused" {
		t.Errorf("Excerpt: %q", got)
	}
	if got, _ := redact.Value([]string{"database"}, in); got != `"`+want+`"` {
		t.Errorf("Value: %q", got)
	}
}
