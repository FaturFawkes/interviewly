#!/usr/bin/env python3
import base64
import hashlib
import hmac
import json
import time
import urllib.error
import urllib.request


class E2EError(Exception):
    pass


def encode_urlsafe_json(data: dict) -> str:
    payload = json.dumps(data, separators=(",", ":")).encode()
    return base64.urlsafe_b64encode(payload).rstrip(b"=").decode()


def build_jwt(secret: str, issuer: str, user_id: str) -> str:
    header = {"alg": "HS256", "typ": "JWT"}
    claims = {
        "sub": user_id,
        "iss": issuer,
        "exp": int(time.time()) + 3600,
    }
    unsigned = f"{encode_urlsafe_json(header)}.{encode_urlsafe_json(claims)}"
    signature = hmac.new(secret.encode(), unsigned.encode(), hashlib.sha256).digest()
    encoded_signature = base64.urlsafe_b64encode(signature).rstrip(b"=").decode()
    return f"{unsigned}.{encoded_signature}"


def request_json(base_url: str, method: str, path: str, body=None, token: str | None = None):
    headers = {"Content-Type": "application/json"}
    if token is not None:
        headers["Authorization"] = f"Bearer {token}"

    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(base_url + path, data=data, headers=headers, method=method)

    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            raw = resp.read().decode()
            return resp.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as err:
        raw = err.read().decode()
        try:
            parsed = json.loads(raw) if raw else {}
        except json.JSONDecodeError:
            parsed = {"raw": raw}
        return err.code, parsed


def request_text(base_url: str, path: str):
    req = urllib.request.Request(base_url + path, method="GET")
    with urllib.request.urlopen(req, timeout=20) as resp:
        return resp.status, resp.read().decode()


def assert_check(name: str, condition: bool, details: str):
    if not condition:
        raise E2EError(f"{name} failed: {details}")
    print(f"PASS - {name}")


def wait_until_ready(base_url: str, timeout_sec: int = 90):
    deadline = time.time() + timeout_sec
    while time.time() < deadline:
        try:
            status, _ = request_text(base_url, "/")
            if status == 200:
                return
        except Exception:
            pass
        time.sleep(1.0)
    raise E2EError("frontend service did not become ready in time")


def main() -> int:
    base_url = "http://127.0.0.1:3000"
    token = build_jwt("compose-secret", "compose-issuer", "compose-user")

    wait_until_ready(base_url)

    status, html = request_text(base_url, "/")
    assert_check("Landing page reachable", status == 200 and "AI Interview Coach" in html, f"status={status}")

    status, health = request_json(base_url, "GET", "/api-proxy/health")
    assert_check("Backend health via frontend proxy", status == 200 and health.get("status") == "ok", f"status={status} body={health}")

    status, me = request_json(base_url, "GET", "/api-proxy/api/me", token=token)
    assert_check("Authenticated /api/me via proxy", status == 200 and me.get("id") == "compose-user", f"status={status} body={me}")

    status, generated = request_json(
        base_url,
        "POST",
        "/api-proxy/api/questions/generate",
        {
            "resume_text": "I built scalable APIs in Go and improved p95 latency by 40%",
            "job_description": "Seeking backend engineer with Go, PostgreSQL, Redis and system design",
        },
        token=token,
    )
    questions = generated.get("questions", [])
    assert_check(
        "Generate interview questions",
        status == 200 and len(questions) == 10,
        f"status={status} count={len(questions)} body={generated}",
    )

    resume_id = generated.get("resume_id", "")
    job_parse_id = generated.get("job_parse_id", "")
    question_ids = [question["id"] for question in questions]
    assert_check("Question response IDs", bool(resume_id and job_parse_id), f"resume_id={resume_id} job_parse_id={job_parse_id}")

    status, session = request_json(
        base_url,
        "POST",
        "/api-proxy/api/session/start",
        {
            "resume_id": resume_id,
            "job_parse_id": job_parse_id,
            "question_ids": question_ids,
        },
        token=token,
    )
    session_id = session.get("id", "")
    assert_check("Start interview session", status == 200 and bool(session_id), f"status={status} body={session}")

    status, answer = request_json(
        base_url,
        "POST",
        "/api-proxy/api/session/answer",
        {
            "session_id": session_id,
            "question_id": question_ids[0],
            "answer": "I used STAR to align architecture decisions and reduced incidents by 30%.",
        },
        token=token,
    )
    assert_check("Submit interview answer", status == 200 and bool(answer.get("id")), f"status={status} body={answer}")

    status, feedback = request_json(
        base_url,
        "POST",
        "/api-proxy/api/feedback/generate",
        {
            "session_id": session_id,
            "question_id": question_ids[0],
            "question": questions[0]["question"],
            "answer": answer.get("answer", ""),
        },
        token=token,
    )
    assert_check("Generate feedback", status == 200 and feedback.get("score", 0) > 0, f"status={status} body={feedback}")

    status, progress = request_json(base_url, "GET", "/api-proxy/api/progress", token=token)
    assert_check("Fetch progress", status == 200 and "average_score" in progress, f"status={status} body={progress}")

    print("\nE2E compose frontend-backend test succeeded")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
