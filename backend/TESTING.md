# Backend Testing

## Unit/Compile Verification

Run:

```bash
make test
```

## E2E API Regression (FAT-42)

Run:

```bash
make test-e2e
```

What this verifies automatically:

- Positive path: health, auth, question generation, session start, answer submission, feedback generation, session history, and progress.
- Negative path: invalid token and invalid session/question references are rejected.
- Contract safety: fails fast if response shape/status changes unexpectedly.

The script starts the backend in test mode on a random free port and stops it automatically after the run.
