-- CreateEnum
CREATE TYPE "user_state" AS ENUM ('pending_email_verification', 'active', 'blocked', 'deleted');

-- CreateEnum
CREATE TYPE "user_role" AS ENUM ('user', 'moderator', 'admin');

-- CreateEnum
CREATE TYPE "auth_otp_challenge_kind" AS ENUM ('email_verification', 'password_reset');

-- CreateTable
CREATE TABLE "users" (
    "id" UUID NOT NULL,
    "email_canonical" TEXT NOT NULL,
    "email_original" TEXT NOT NULL,
    "nickname" TEXT NOT NULL,
    "display_name" TEXT NOT NULL,
    "password_hash" TEXT NOT NULL,
    "password_algo" TEXT NOT NULL DEFAULT 'argon2id',
    "password_pepper_id" TEXT,
    "state" "user_state" NOT NULL DEFAULT 'pending_email_verification',
    "email_verified_at" TIMESTAMP(3),
    "pending_expires_at" TIMESTAMP(3),
    "role" "user_role" NOT NULL DEFAULT 'user',
    "token_version" INTEGER NOT NULL DEFAULT 0,
    "last_login_at" TIMESTAMP(3),
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "sessions" (
    "id" UUID NOT NULL,
    "user_id" UUID NOT NULL,
    "device_name" TEXT,
    "platform" TEXT,
    "user_agent" TEXT,
    "ip" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "last_seen_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "revoked_at" TIMESTAMP(3),

    CONSTRAINT "sessions_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "refresh_tokens" (
    "id" UUID NOT NULL,
    "session_id" UUID NOT NULL,
    "user_id" UUID NOT NULL,
    "token_hash" TEXT NOT NULL,
    "parent_id" UUID,
    "replaced_by_id" UUID,
    "issued_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "expires_at" TIMESTAMP(3) NOT NULL,
    "revoked_at" TIMESTAMP(3),

    CONSTRAINT "refresh_tokens_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "auth_otp_challenges" (
    "id" UUID NOT NULL,
    "kind" "auth_otp_challenge_kind" NOT NULL,
    "user_id" UUID,
    "email_canonical" TEXT NOT NULL,
    "code_hash" TEXT NOT NULL,
    "attempts" INTEGER NOT NULL DEFAULT 0,
    "resends" INTEGER NOT NULL DEFAULT 0,
    "expires_at" TIMESTAMP(3) NOT NULL,
    "last_resend_at" TIMESTAMP(3),
    "consumed_at" TIMESTAMP(3),
    "invalidated_at" TIMESTAMP(3),
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "auth_otp_challenges_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "audit_events" (
    "id" UUID NOT NULL,
    "event_type" TEXT NOT NULL,
    "user_id" UUID,
    "session_id" UUID,
    "ip" TEXT,
    "user_agent" TEXT,
    "meta" JSONB NOT NULL DEFAULT '{}',
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "audit_events_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "users_state_pending_expires_at_idx" ON "users"("state", "pending_expires_at");

-- CreateIndex
CREATE INDEX "sessions_user_id_revoked_at_idx" ON "sessions"("user_id", "revoked_at");

-- CreateIndex
CREATE UNIQUE INDEX "refresh_tokens_token_hash_key" ON "refresh_tokens"("token_hash");

-- CreateIndex
CREATE INDEX "refresh_tokens_session_id_revoked_at_idx" ON "refresh_tokens"("session_id", "revoked_at");

-- CreateIndex
CREATE INDEX "refresh_tokens_user_id_revoked_at_idx" ON "refresh_tokens"("user_id", "revoked_at");

-- CreateIndex
CREATE INDEX "auth_otp_challenges_kind_user_id_consumed_at_invalidated_at_idx" ON "auth_otp_challenges"("kind", "user_id", "consumed_at", "invalidated_at");

-- CreateIndex
CREATE INDEX "auth_otp_challenges_email_canonical_kind_idx" ON "auth_otp_challenges"("email_canonical", "kind");

-- CreateIndex
CREATE INDEX "audit_events_user_id_created_at_idx" ON "audit_events"("user_id", "created_at");

-- CreateIndex
CREATE INDEX "audit_events_event_type_created_at_idx" ON "audit_events"("event_type", "created_at");

-- AddForeignKey
ALTER TABLE "sessions" ADD CONSTRAINT "sessions_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "refresh_tokens" ADD CONSTRAINT "refresh_tokens_session_id_fkey" FOREIGN KEY ("session_id") REFERENCES "sessions"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "refresh_tokens" ADD CONSTRAINT "refresh_tokens_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "refresh_tokens" ADD CONSTRAINT "refresh_tokens_parent_id_fkey" FOREIGN KEY ("parent_id") REFERENCES "refresh_tokens"("id") ON DELETE NO ACTION ON UPDATE NO ACTION;

-- AddForeignKey
ALTER TABLE "auth_otp_challenges" ADD CONSTRAINT "auth_otp_challenges_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "audit_events" ADD CONSTRAINT "audit_events_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- Partial unique indexes — `email_canonical` and `nickname` are unique only
-- among non-deleted users. Tombstoned (state='deleted') rows free up their
-- email/nickname for re-registration. Prisma can't emit partial unique
-- indexes, so they live here in raw SQL.
CREATE UNIQUE INDEX "users_email_canonical_active_uq"
  ON "users" ("email_canonical")
  WHERE "state" <> 'deleted';

CREATE UNIQUE INDEX "users_nickname_active_uq"
  ON "users" ("nickname")
  WHERE "state" <> 'deleted';
