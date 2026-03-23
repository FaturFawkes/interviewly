#!/usr/bin/env python3
import base64
import hashlib
import hmac
import json
import os
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request


class E2EError(Exception):
    pass


def find_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


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
        with urllib.request.urlopen(req, timeout=10) as resp:
            raw = resp.read().decode()
            return resp.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as err:
        raw = err.read().decode()
        try:
            parsed = json.loads(raw) if raw else {}
        except json.JSONDecodeError:
            parsed = {"raw": raw}
        return err.code, parsed


def assert_check(name: str, condition: bool, details: str):
    if not condition:
        raise E2EError(f"{name} failed: {details}")
    print(f"PASS - {name}")


def wait_until_healthy(base_url: str, timeout_sec: int = 25):
    deadline = time.time() + timeout_sec
    while time.time() < deadline:
        try:
            status, body = request_json(base_url, "GET", "/health", token=None)
            if status == 200 and body.get("status") == "ok":
                return
        except Exception:
            pass
        time.sleep(0.3)
    raise E2EError("server did not become healthy in time")


def read_log_tail(log_path: str, lines: int = 40) -> str:
    try:
        with open(log_path, "r", encoding="utf-8") as file:
            content = file.readlines()
        return "".join(content[-lines:])
    except FileNotFoundError:
        return "<log file missing>"


def stop_server(process: subprocess.Popen):
    if process.poll() is not None:
        return

    process.terminate()
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait(timeout=5)


def main() -> int:
    backend_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    port = find_free_port()
    base_url = f"http://127.0.0.1:{port}"

    env = os.environ.copy()
    env["APP_ENV"] = env.get("APP_ENV", "test")
    env["PORT"] = str(port)
    env["JWT_SECRET"] = env.get("JWT_SECRET", "e2e-secret")
    env["JWT_ISSUER"] = env.get("JWT_ISSUER", "e2e-issuer")

    user_id = "user-e2e-1"
    token = build_jwt(env["JWT_SECRET"], env["JWT_ISSUER"], user_id)

    with tempfile.NamedTemporaryFile(prefix="backend-e2e-", suffix=".log", delete=False) as file:
        log_path = file.name

    log_file = open(log_path, "w", encoding="utf-8")
    process = subprocess.Popen(
        ["go", "run", "./cmd/server"],
        cwd=backend_root,
        env=env,
        stdout=log_file,
        stderr=subprocess.STDOUT,
    )

    try:
        wait_until_healthy(base_url)

        status, health = request_json(base_url, "GET", "/health", token=None)
        assert_check("GET /health", status == 200 and health.get("status") == "ok", f"status={status} body={health}")

        status, unauthorized = request_json(base_url, "GET", "/api/me", token=None)
        assert_check("GET /api/me unauthorized", status == 401, f"status={status} body={unauthorized}")

        status, me = request_json(base_url, "GET", "/api/me", token=token)
        assert_check("GET /api/me", status == 200 and me.get("id") == user_id, f"status={status} body={me}")

        status, generated = request_json(
            base_url,
            "POST",
            "/api/questions/generate",
            {
                "resume_text": "I built production Go microservices and improved API latency by 40 percent",
                "job_description": "Hiring backend Go engineer for scalable APIs with PostgreSQL and Redis",
            },
            token=token,
        )
        questions = generated.get("questions", [])
        assert_check(
            "POST /api/questions/generate",
            status == 200 and len(questions) == 10,
            f"status={status} count={len(questions)} body={generated}",
        )

        resume_id = generated.get("resume_id", "")
        job_parse_id = generated.get("job_parse_id", "")
        assert_check(
            "questions response ids",
            bool(resume_id) and bool(job_parse_id),
            f"resume_id={resume_id} job_parse_id={job_parse_id}",
        )

        question_ids = [question["id"] for question in questions]

        status, session = request_json(
            base_url,
            "POST",
            "/api/session/start",
            {
                "resume_id": resume_id,
                "job_parse_id": job_parse_id,
                "question_ids": question_ids,
            },
            token=token,
        )
        session_id = session.get("id", "")
        assert_check(
            "POST /api/session/start",
            status == 200 and bool(session_id),
            f"status={status} body={session}",
        )

        status, answer = request_json(
            base_url,
            "POST",
            "/api/session/answer",
            {
                "session_id": session_id,
                "question_id": question_ids[0],
                "answer": "I led a migration rollout, coordinated stakeholders, and reduced production incidents by 35 percent.",
            },
            token=token,
        )
        assert_check(
            "POST /api/session/answer",
            status == 200 and bool(answer.get("id")),
            f"status={status} body={answer}",
        )

        status, feedback = request_json(
            base_url,
            "POST",
            "/api/feedback/generate",
            {
                "session_id": session_id,
                "question_id": question_ids[0],
                "question": questions[0]["question"],
                "answer": answer.get("answer", ""),
            },
            token=token,
        )
        assert_check(
            "POST /api/feedback/generate",
            status == 200 and feedback.get("score", 0) > 0,
            f"status={status} body={feedback}",
        )

        status, history = request_json(base_url, "GET", "/api/session/history", token=token)
        assert_check(
            "GET /api/session/history",
            status == 200 and any(item.get("id") == session_id for item in history.get("sessions", [])),
            f"status={status} body={history}",
        )

        status, progress = request_json(base_url, "GET", "/api/progress", token=token)
        assert_check(
            "GET /api/progress",
            status == 200
            and "interview_progress" in progress
            and "review_progress" in progress
            and "average_score" in progress.get("interview_progress", {}),
            f"status={status} body={progress}",
        )

        invalid_token = "invalid.token.value"
        status, invalid_me = request_json(base_url, "GET", "/api/me", token=invalid_token)
        assert_check(
            "GET /api/me invalid token",
            status == 401,
            f"status={status} body={invalid_me}",
        )

        status, invalid_answer = request_json(
            base_url,
            "POST",
            "/api/session/answer",
            {
                "session_id": "not-exist",
                "question_id": "not-exist",
                "answer": "test",
            },
            token=token,
        )
        assert_check(
            "POST /api/session/answer invalid refs",
            status == 400,
            f"status={status} body={invalid_answer}",
        )

        status, invalid_feedback = request_json(
            base_url,
            "POST",
            "/api/feedback/generate",
            {
                "session_id": "not-exist",
                "question_id": "not-exist",
                "question": "Tell me about yourself",
                "answer": "test",
            },
            token=token,
        )
        assert_check(
            "POST /api/feedback/generate invalid refs",
            status == 400,
            f"status={status} body={invalid_feedback}",
        )

        # --- Resume ---
        status, resume = request_json(
            base_url,
            "POST",
            "/api/resume",
            {"content": "Experienced Go backend engineer with 5 years building microservices."},
            token=token,
        )
        assert_check(
            "POST /api/resume",
            status == 200 and bool(resume.get("id")),
            f"status={status} body={resume}",
        )

        status, latest_resume = request_json(base_url, "GET", "/api/resume", token=token)
        assert_check(
            "GET /api/resume",
            status == 200 and bool(latest_resume.get("id")),
            f"status={status} body={latest_resume}",
        )

        # --- Job parse ---
        status, parsed_job = request_json(
            base_url,
            "POST",
            "/api/job/parse",
            {"job_description": "Hiring senior Go engineer for scalable APIs with PostgreSQL and Redis."},
            token=token,
        )
        assert_check(
            "POST /api/job/parse",
            status == 200 and bool(parsed_job.get("id")),
            f"status={status} body={parsed_job}",
        )

        # --- Subscription status ---
        status, sub_status = request_json(base_url, "GET", "/api/subscription/status", token=token)
        assert_check(
            "GET /api/subscription/status",
            status == 200
            and "plan_id" in sub_status
            and "remaining_sessions" in sub_status
            and "remaining_voice_minutes" in sub_status,
            f"status={status} body={sub_status}",
        )

        # --- Analytics overview ---
        status, analytics = request_json(base_url, "GET", "/api/analytics/overview", token=token)
        assert_check(
            "GET /api/analytics/overview",
            status == 200 and "average_score" in analytics and "sessions_completed" in analytics,
            f"status={status} body={analytics}",
        )

        # --- Review session flow ---
        status, review_start = request_json(
            base_url,
            "POST",
            "/api/review/start",
            {
                "session_type": "review",
                "input_mode": "text",
                "interview_language": "en",
                "input_text": "I was asked about a conflict I handled. I explained the situation but skipped the result.",
                "interview_prompt": "Tell me about a conflict you handled at work.",
                "target_role": "Backend Engineer",
                "target_company": "Tech Corp",
            },
            token=token,
        )
        review_session_id = review_start.get("session", {}).get("id", "")
        assert_check(
            "POST /api/review/start",
            status == 200 and bool(review_session_id) and "feedback" in review_start,
            f"status={status} body={review_start}",
        )

        status, review_respond = request_json(
            base_url,
            "POST",
            "/api/review/respond",
            {
                "session_id": review_session_id,
                "input_text": "The conflict was about deployment timelines. I mediated between the team and PM, and we shipped on time.",
            },
            token=token,
        )
        assert_check(
            "POST /api/review/respond",
            status == 200 and "feedback" in review_respond,
            f"status={status} body={review_respond}",
        )

        status, review_end = request_json(
            base_url,
            "POST",
            "/api/review/end",
            {"session_id": review_session_id},
            token=token,
        )
        assert_check(
            "POST /api/review/end",
            status == 200
            and "improvement_plan" in review_end
            and "coaching_summary" in review_end,
            f"status={status} body={review_end}",
        )

        # --- Coaching summary ---
        status, coaching = request_json(base_url, "GET", "/api/coaching-summary", token=token)
        assert_check(
            "GET /api/coaching-summary",
            status == 200
            and "feedback" in coaching
            and "improvement_plan" in coaching
            and "session_id" in coaching,
            f"status={status} body={coaching}",
        )

        # --- Error: review respond with invalid session ---
        status, invalid_respond = request_json(
            base_url,
            "POST",
            "/api/review/respond",
            {"session_id": "not-exist", "input_text": "test"},
            token=token,
        )
        assert_check(
            "POST /api/review/respond invalid session",
            status == 400,
            f"status={status} body={invalid_respond}",
        )

        print("\nE2E regression succeeded")
        print(f"Server log: {log_path}")
        return 0
    except E2EError as err:
        print(f"\nE2E regression failed: {err}", file=sys.stderr)
        print(f"\n--- Server log tail ({log_path}) ---", file=sys.stderr)
        print(read_log_tail(log_path), file=sys.stderr)
        return 1
    finally:
        stop_server(process)
        log_file.close()


if __name__ == "__main__":
    sys.exit(main())
