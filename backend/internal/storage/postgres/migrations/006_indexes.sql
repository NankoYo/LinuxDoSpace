-- 006_indexes.sql adds PostgreSQL-oriented indexes for the query patterns that
-- grow the fastest in production: allocation ownership lists, admin review
-- pages, email target sorting, and the public supervision audit-log scan.

CREATE INDEX IF NOT EXISTS idx_users_last_login_at_id
ON users(last_login_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_managed_domains_enabled_default_root
ON managed_domains(enabled, is_default DESC, root_domain ASC);

CREATE INDEX IF NOT EXISTS idx_allocations_user_domain_status
ON allocations(user_id, managed_domain_id, status);

CREATE INDEX IF NOT EXISTS idx_allocations_user_status_sort
ON allocations(user_id, status, is_primary DESC, created_at ASC);

CREATE INDEX IF NOT EXISTS idx_allocations_status_created_at
ON allocations(status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_email_routes_owner_created_at
ON email_routes(owner_user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_applications_applicant_updated_at
ON admin_applications(applicant_user_id, updated_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_email_targets_owner_sort
ON email_targets(
    owner_user_id,
    (CASE WHEN verified_at IS NULL THEN 1 ELSE 0 END),
    updated_at DESC,
    id DESC
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_dns_record_latest
ON audit_logs(resource_id, created_at DESC, id DESC)
WHERE resource_type = 'dns_record'
  AND action IN ('dns_record.create', 'dns_record.update', 'dns_record.delete')
  AND NULLIF(metadata_json::jsonb ->> 'allocation_id', '') IS NOT NULL;
