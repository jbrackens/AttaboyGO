-- 000008_phase8_affiliates.up.sql
-- Full affiliate system: 19 tables

CREATE TABLE affiliate_brands (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    code        varchar(10)  NOT NULL UNIQUE,
    name        varchar(200) NOT NULL,
    active      boolean      NOT NULL DEFAULT true,
    created_at  timestamp    DEFAULT now()
);

CREATE TABLE affiliates (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    email               varchar(300) NOT NULL UNIQUE,
    password_hash       varchar(200) NOT NULL,
    first_name          varchar(100),
    last_name           varchar(100),
    company             varchar(200),
    status              varchar(30)  NOT NULL DEFAULT 'pending',
    affiliate_code      varchar(50)  NOT NULL UNIQUE,
    parent_affiliate_id uuid         REFERENCES affiliates(id) ON DELETE SET NULL,
    tier                varchar(30)  NOT NULL DEFAULT 'standard',
    country             varchar(2),
    phone               varchar(30),
    website             varchar(500),
    payment_method      varchar(50),
    payment_details     jsonb        DEFAULT '{}',
    created_at          timestamp    DEFAULT now(),
    updated_at          timestamp    DEFAULT now()
);

CREATE INDEX idx_affiliates_status ON affiliates (status);
CREATE INDEX idx_affiliates_affiliate_code ON affiliates (affiliate_code);

CREATE TABLE affiliate_tokens (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id uuid        NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    token        varchar(500) NOT NULL,
    expires_at   timestamp    NOT NULL,
    created_at   timestamp    DEFAULT now()
);

CREATE INDEX idx_affiliate_tokens_affiliate_id ON affiliate_tokens (affiliate_id);
CREATE INDEX idx_affiliate_tokens_token ON affiliate_tokens (token);

CREATE TABLE affiliate_plans (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name         varchar(200) NOT NULL,
    description  text,
    type         varchar(30)  NOT NULL,
    status       varchar(30)  NOT NULL DEFAULT 'active',
    default_plan boolean      NOT NULL DEFAULT false,
    created_at   timestamp    DEFAULT now(),
    updated_at   timestamp    DEFAULT now()
);

CREATE TABLE affiliate_plan_tiers (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id          uuid        NOT NULL REFERENCES affiliate_plans(id) ON DELETE CASCADE,
    tier_level       integer     NOT NULL,
    threshold_amount integer     NOT NULL DEFAULT 0,
    cpa_amount_minor integer     NOT NULL DEFAULT 0,
    rev_share_pct    decimal(5,2) NOT NULL DEFAULT 0,
    created_at       timestamp   DEFAULT now()
);

CREATE INDEX idx_affiliate_plan_tiers_plan_id ON affiliate_plan_tiers (plan_id);

CREATE TABLE affiliate_deals (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id         uuid        NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    plan_id              uuid        NOT NULL REFERENCES affiliate_plans(id) ON DELETE RESTRICT,
    brand                varchar(10) NOT NULL,
    status               varchar(30) NOT NULL DEFAULT 'active',
    custom_cpa_minor     integer,
    custom_rev_share_pct decimal(5,2),
    started_at           timestamp   DEFAULT now(),
    ended_at             timestamp,
    created_at           timestamp   DEFAULT now(),
    updated_at           timestamp   DEFAULT now()
);

CREATE INDEX idx_affiliate_deals_affiliate_id ON affiliate_deals (affiliate_id);
CREATE INDEX idx_affiliate_deals_affiliate_brand ON affiliate_deals (affiliate_id, brand);

CREATE TABLE affiliate_links (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id   uuid        NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    deal_id        uuid        REFERENCES affiliate_deals(id) ON DELETE SET NULL,
    btag           varchar(100) NOT NULL UNIQUE,
    target_url     varchar(1000) NOT NULL,
    campaign       varchar(200),
    medium         varchar(100),
    content        varchar(200),
    clicks         integer      NOT NULL DEFAULT 0,
    registrations  integer      NOT NULL DEFAULT 0,
    active         boolean      NOT NULL DEFAULT true,
    created_at     timestamp    DEFAULT now(),
    updated_at     timestamp    DEFAULT now()
);

CREATE INDEX idx_affiliate_links_affiliate_id ON affiliate_links (affiliate_id);
CREATE INDEX idx_affiliate_links_btag ON affiliate_links (btag);

CREATE TABLE affiliate_clicks (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id      uuid        NOT NULL REFERENCES affiliate_links(id) ON DELETE CASCADE,
    ip_address   varchar(45),
    user_agent   text,
    country      varchar(2),
    segment      varchar(100),
    referrer_url varchar(1000),
    clicked_at   timestamp   DEFAULT now()
);

CREATE INDEX idx_affiliate_clicks_link_id ON affiliate_clicks (link_id);
CREATE INDEX idx_affiliate_clicks_clicked_at ON affiliate_clicks (clicked_at);

CREATE TABLE affiliate_player_refs (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id    uuid        NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    link_id         uuid        REFERENCES affiliate_links(id) ON DELETE SET NULL,
    player_id       uuid        NOT NULL REFERENCES v2_players(id) ON DELETE CASCADE,
    registered_at   timestamp   DEFAULT now(),
    first_deposit_at timestamp,
    status          varchar(30) NOT NULL DEFAULT 'active',
    created_at      timestamp   DEFAULT now(),
    UNIQUE (affiliate_id, player_id)
);

CREATE INDEX idx_affiliate_player_refs_affiliate_id ON affiliate_player_refs (affiliate_id);
CREATE INDEX idx_affiliate_player_refs_player_id ON affiliate_player_refs (player_id);

CREATE TABLE affiliate_fees (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_id     uuid        NOT NULL REFERENCES affiliate_deals(id) ON DELETE CASCADE,
    fee_type    varchar(50) NOT NULL,
    name        varchar(200) NOT NULL,
    calc_method varchar(30) NOT NULL,
    value_minor integer     NOT NULL DEFAULT 0,
    value_pct   decimal(5,2) NOT NULL DEFAULT 0,
    created_at  timestamp   DEFAULT now()
);

CREATE INDEX idx_affiliate_fees_deal_id ON affiliate_fees (deal_id);

CREATE TABLE affiliate_commissions (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id        uuid        NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    deal_id             uuid        NOT NULL REFERENCES affiliate_deals(id) ON DELETE CASCADE,
    period_start        date        NOT NULL,
    period_end          date        NOT NULL,
    gross_revenue_minor integer     NOT NULL DEFAULT 0,
    total_fees_minor    integer     NOT NULL DEFAULT 0,
    net_revenue_minor   integer     NOT NULL DEFAULT 0,
    commission_minor    integer     NOT NULL DEFAULT 0,
    currency            varchar(3)  NOT NULL DEFAULT 'EUR',
    status              varchar(30) NOT NULL DEFAULT 'draft',
    player_count        integer     NOT NULL DEFAULT 0,
    ftd_count           integer     NOT NULL DEFAULT 0,
    created_at          timestamp   DEFAULT now(),
    updated_at          timestamp   DEFAULT now()
);

CREATE INDEX idx_affiliate_commissions_affiliate_id ON affiliate_commissions (affiliate_id);
CREATE INDEX idx_affiliate_commissions_affiliate_period ON affiliate_commissions (affiliate_id, period_start, period_end);

CREATE TABLE affiliate_invoices (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id    uuid        NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    invoice_number  varchar(50) NOT NULL UNIQUE,
    period_start    date        NOT NULL,
    period_end      date        NOT NULL,
    amount_minor    integer     NOT NULL DEFAULT 0,
    currency        varchar(3)  NOT NULL DEFAULT 'EUR',
    status          varchar(30) NOT NULL DEFAULT 'draft',
    confirmed_at    timestamp,
    paid_at         timestamp,
    created_at      timestamp   DEFAULT now(),
    updated_at      timestamp   DEFAULT now()
);

CREATE INDEX idx_affiliate_invoices_affiliate_id ON affiliate_invoices (affiliate_id);
CREATE INDEX idx_affiliate_invoices_invoice_number ON affiliate_invoices (invoice_number);

CREATE TABLE affiliate_invoice_lines (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id     uuid        NOT NULL REFERENCES affiliate_invoices(id) ON DELETE CASCADE,
    commission_id  uuid        REFERENCES affiliate_commissions(id) ON DELETE SET NULL,
    description    varchar(500) NOT NULL,
    amount_minor   integer     NOT NULL,
    created_at     timestamp   DEFAULT now()
);

CREATE INDEX idx_affiliate_invoice_lines_invoice_id ON affiliate_invoice_lines (invoice_id);

CREATE TABLE affiliate_payments (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id uuid        NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    invoice_id   uuid        REFERENCES affiliate_invoices(id) ON DELETE SET NULL,
    amount_minor integer     NOT NULL,
    currency     varchar(3)  NOT NULL DEFAULT 'EUR',
    method       varchar(50) NOT NULL,
    reference    varchar(200),
    status       varchar(30) NOT NULL DEFAULT 'pending',
    paid_at      timestamp,
    created_at   timestamp   DEFAULT now(),
    updated_at   timestamp   DEFAULT now()
);

CREATE INDEX idx_affiliate_payments_affiliate_id ON affiliate_payments (affiliate_id);
CREATE INDEX idx_affiliate_payments_invoice_id ON affiliate_payments (invoice_id);

CREATE TABLE affiliate_reports_cache (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id uuid        REFERENCES affiliates(id) ON DELETE CASCADE,
    report_type  varchar(50) NOT NULL,
    period       varchar(20) NOT NULL,
    data         jsonb       NOT NULL DEFAULT '{}',
    generated_at timestamp   DEFAULT now()
);

CREATE INDEX idx_affiliate_reports_cache_lookup ON affiliate_reports_cache (affiliate_id, report_type, period);

CREATE TABLE affiliate_notifications (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id uuid        NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    type         varchar(50) NOT NULL,
    title        varchar(300) NOT NULL,
    message      text,
    read         boolean     NOT NULL DEFAULT false,
    created_at   timestamp   DEFAULT now()
);

CREATE INDEX idx_affiliate_notifications_affiliate_id ON affiliate_notifications (affiliate_id);
CREATE INDEX idx_affiliate_notifications_affiliate_read ON affiliate_notifications (affiliate_id, read);

CREATE TABLE affiliate_media (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        varchar(200) NOT NULL,
    type        varchar(50) NOT NULL,
    dimensions  varchar(30),
    url         varchar(1000) NOT NULL,
    brand       varchar(10),
    active      boolean     NOT NULL DEFAULT true,
    created_at  timestamp   DEFAULT now(),
    updated_at  timestamp   DEFAULT now()
);

CREATE TABLE affiliate_sub_affiliates (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id            uuid        NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    child_id             uuid        NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    commission_share_pct decimal(5,2) NOT NULL DEFAULT 0,
    created_at           timestamp   DEFAULT now(),
    UNIQUE (parent_id, child_id)
);

CREATE INDEX idx_affiliate_sub_affiliates_parent_id ON affiliate_sub_affiliates (parent_id);
CREATE INDEX idx_affiliate_sub_affiliates_child_id ON affiliate_sub_affiliates (child_id);

CREATE TABLE affiliate_activity_log (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id uuid        REFERENCES affiliates(id) ON DELETE SET NULL,
    admin_id     uuid        REFERENCES admin_users(id) ON DELETE SET NULL,
    action       varchar(100) NOT NULL,
    details      jsonb        DEFAULT '{}',
    created_at   timestamp    DEFAULT now()
);

CREATE INDEX idx_affiliate_activity_log_affiliate_id ON affiliate_activity_log (affiliate_id);
CREATE INDEX idx_affiliate_activity_log_created_at ON affiliate_activity_log (created_at);
