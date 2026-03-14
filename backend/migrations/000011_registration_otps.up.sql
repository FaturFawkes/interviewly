CREATE TABLE IF NOT EXISTS registration_otps (
    email VARCHAR(255) PRIMARY KEY,
    full_name VARCHAR(255),
    password_hash VARCHAR(255) NOT NULL,
    otp_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_registration_otps_expires_at ON registration_otps(expires_at);
