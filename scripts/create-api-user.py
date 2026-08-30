#!/usr/bin/env python3
"""
Idempotently provisions an Umbraco API user with client-credentials against a local
testsite/, via an admin cookie login + PKCE authorization_code dance against Umbraco's
built-in `umbraco-swagger` OAuth client.

Umbraco's backoffice login cookie is always marked `Secure`, so the admin-authenticated
part of this dance (login/authorize/code-exchange) must run over HTTPS even though the
testsite itself is HTTP-only for everyday CLI use (client_credentials calls don't use
cookies, so they're unaffected and stay on plain HTTP). The dev HTTPS cert is
self-signed and not trusted by the system store in this environment, so TLS
verification is disabled for just this admin flow — same "insecure local dev" trade-off
this project's own `--insecure` profile flag already documents.

Usage:
  python3 scripts/create-api-user.py \
      [--base-url http://localhost:58880] [--admin-base-url https://localhost:58881] \
      [--admin-email admin@admin.com] [--admin-password 1234567890] \
      [--client-id umbraco-cli-testsite] [--client-secret umbraco-cli-testsite-secret]
"""
import argparse
import base64
import hashlib
import http.cookiejar
import json
import secrets
import ssl
import urllib.error
import urllib.parse
import urllib.request

ADMIN_GROUP_ID = "e5e7f6c8-7f9c-4b5b-8d5d-9e1e5a4f7e4d"  # built-in "Administrators" group
API_BASE = "/umbraco/management/api/v1"

INSECURE_SSL_CONTEXT = ssl.create_default_context()
INSECURE_SSL_CONTEXT.check_hostname = False
INSECURE_SSL_CONTEXT.verify_mode = ssl.CERT_NONE


def b64url(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode()


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None  # don't follow — caller reads Location from the resulting HTTPError


def request(opener, method, url, *, data=None, headers=None, expect=(200, 201, 204, 302)):
    headers = dict(headers or {})
    body = None
    if data is not None:
        body = json.dumps(data).encode() if not isinstance(data, (bytes, str)) else data
        if isinstance(body, str):
            body = body.encode()
        headers.setdefault("Content-Type", "application/json")
    req = urllib.request.Request(url, data=body, headers=headers, method=method)
    try:
        resp = opener.open(req)
        status = resp.status
    except urllib.error.HTTPError as e:
        resp = e
        status = e.code
    if status not in expect:
        raise SystemExit(f"{method} {url} -> {status}: {resp.read()[:500]}")
    return resp


def token_works(base_url, client_id, client_secret):
    opener = urllib.request.build_opener()
    form = urllib.parse.urlencode({
        "grant_type": "client_credentials",
        "client_id": client_id,
        "client_secret": client_secret,
    }).encode()
    req = urllib.request.Request(
        base_url + API_BASE + "/security/back-office/token",
        data=form,
        headers={"Content-Type": "application/x-www-form-urlencoded"},
        method="POST",
    )
    try:
        with opener.open(req) as resp:
            return resp.status == 200
    except urllib.error.HTTPError:
        return False


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base-url", default="http://localhost:58880",
                     help="Plain-HTTP base the CLI itself will use (client_credentials only, no cookies)")
    ap.add_argument("--admin-base-url", default="https://localhost:58881",
                     help="HTTPS base for the admin cookie login + PKCE dance (Umbraco's backoffice cookie requires HTTPS)")
    ap.add_argument("--admin-email", default="admin@admin.com")
    ap.add_argument("--admin-password", default="1234567890")
    ap.add_argument("--client-id", default="umbraco-cli-testsite")
    ap.add_argument("--client-secret", default="umbraco-cli-testsite-secret")
    ap.add_argument("--user-name", default="umbraco-cli")
    ap.add_argument("--user-email", default="umbraco-cli@localhost")
    args = ap.parse_args()

    base = args.base_url.rstrip("/")
    admin_base = args.admin_base_url.rstrip("/")
    # Umbraco registers the OpenIddict application with this prefix regardless of what
    # clientId is passed to POST /user/{id}/client-credentials — confirmed empirically
    # against a live instance (also matches the client-id convention already documented
    # in this project's own README/SKILL.md, e.g. "umbraco-back-office-umbraco-cli").
    full_client_id = f"umbraco-back-office-{args.client_id}"

    if token_works(base, full_client_id, args.client_secret):
        print(f"Client credentials for {full_client_id!r} already work — nothing to do.")
        return

    cookiejar = http.cookiejar.CookieJar()
    opener = urllib.request.build_opener(
        urllib.request.HTTPSHandler(context=INSECURE_SSL_CONTEXT),
        urllib.request.HTTPCookieProcessor(cookiejar),
        NoRedirect(),
    )

    print("Logging in as admin (cookie auth, HTTPS)...")
    request(opener, "POST", admin_base + API_BASE + "/security/back-office/login",
            data={"username": args.admin_email, "password": args.admin_password},
            expect=(200,))

    verifier = b64url(secrets.token_bytes(32))
    challenge = b64url(hashlib.sha256(verifier.encode()).digest())
    redirect_uri = admin_base + "/umbraco/openapi/oauth2-redirect.html"

    authorize_qs = urllib.parse.urlencode({
        "client_id": "umbraco-swagger",
        "redirect_uri": redirect_uri,
        "response_type": "code",
        "code_challenge": challenge,
        "code_challenge_method": "S256",
    })

    print("Requesting authorization code (PKCE)...")
    resp = request(opener, "GET", admin_base + API_BASE + f"/security/back-office/authorize?{authorize_qs}",
                   expect=(302,))
    location = resp.headers.get("Location")
    if not location:
        raise SystemExit("Authorize endpoint did not return a redirect Location header.")
    code = urllib.parse.parse_qs(urllib.parse.urlparse(location).query).get("code", [None])[0]
    if not code:
        raise SystemExit(f"No 'code' in redirect: {location}")

    print("Exchanging code for bearer token...")
    form = urllib.parse.urlencode({
        "grant_type": "authorization_code",
        "client_id": "umbraco-swagger",
        "redirect_uri": redirect_uri,
        "code": code,
        "code_verifier": verifier,
    }).encode()
    token_resp = request(opener, "POST", admin_base + API_BASE + "/security/back-office/token",
                         data=form, headers={"Content-Type": "application/x-www-form-urlencoded"},
                         expect=(200,))
    bearer = json.loads(token_resp.read())["access_token"]
    auth_header = {"Authorization": f"Bearer {bearer}"}

    print(f"Creating API user {args.user_name!r}...")
    user_resp = request(opener, "POST", admin_base + API_BASE + "/user",
                        data={
                            "userName": args.user_email,
                            "email": args.user_email,
                            "name": args.user_name,
                            "userGroupIds": [{"id": ADMIN_GROUP_ID}],
                            "kind": "Api",
                        },
                        headers=auth_header, expect=(201,))
    user_id = user_resp.headers.get("umb-generated-resource")
    if not user_id:
        loc = user_resp.headers.get("Location", "")
        user_id = loc.rstrip("/").rsplit("/", 1)[-1]
    if not user_id:
        raise SystemExit("Could not determine new user id from response headers.")

    print(f"Setting client-credentials on user {user_id}...")
    request(opener, "POST", admin_base + API_BASE + f"/user/{user_id}/client-credentials",
            data={"clientId": args.client_id, "clientSecret": args.client_secret},
            headers=auth_header, expect=(200, 201, 204))

    if not token_works(base, full_client_id, args.client_secret):
        raise SystemExit("Provisioned user but client_credentials still don't work — check server logs.")

    print("Done. Wire it in with:")
    print(f"  umbraco auth add testsite --base-url {base} "
          f"--client-id {full_client_id} --client-secret {args.client_secret}")


if __name__ == "__main__":
    main()
