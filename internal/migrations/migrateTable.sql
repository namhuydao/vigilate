CREATE TABLE users
(
    id           SERIAL
        PRIMARY KEY,
    first_name   VARCHAR(255)        NOT NULL,
    last_name    VARCHAR(255)        NOT NULL,
    user_active  INTEGER   DEFAULT 0 NOT NULL,
    access_level INTEGER   DEFAULT 3 NOT NULL,
    email        VARCHAR(255)        NOT NULL,
    password     VARCHAR(60)         NOT NULL,
    deleted_at   TIMESTAMP,
    created_at   TIMESTAMP DEFAULT NOW() NOT NULL,
    updated_at   TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE TABLE preferences
(
    id         SERIAL
        PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    preference TEXT         NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE remember_tokens
(
    id             SERIAL
        PRIMARY KEY,
    user_id        INTEGER      NULL
        CONSTRAINT remember_tokens_users_id_fk
            REFERENCES users
            ON UPDATE CASCADE ON DELETE SET NULL,
    remember_token VARCHAR(100) NOT NULL,
    created_at     TIMESTAMP DEFAULT NOW(),
    updated_at     TIMESTAMP DEFAULT NOW()
);

CREATE TABLE hosts
(
    id             SERIAL
        PRIMARY KEY,
    host_name      VARCHAR(255)      NOT NULL,
    canonical_name VARCHAR(255)      NOT NULL,
    url            VARCHAR(255)      NOT NULL,
    ip             VARCHAR(255)      NOT NULL,
    ipv6           VARCHAR(255)      NOT NULL,
    location       VARCHAR(255)      NOT NULL,
    os             VARCHAR(255)      NOT NULL,
    active         INTEGER DEFAULT 1 NOT NULL,
    created_at     TIMESTAMP         NOT NULL,
    updated_at     TIMESTAMP         NOT NULL
);

CREATE TABLE services
(
    id           SERIAL
        PRIMARY KEY,
    service_name VARCHAR(255)      NOT NULL,
    active       INTEGER DEFAULT 1 NOT NULL,
    icon         VARCHAR(255)      NOT NULL,
    created_at   TIMESTAMP         NOT NULL,
    updated_at   TIMESTAMP         NOT NULL
);

CREATE TABLE host_services
(
    id              SERIAL
        PRIMARY KEY,
    host_id         INTEGER                                                                 NOT NULL
        CONSTRAINT host_services_hosts_id_fk
            REFERENCES hosts
            ON UPDATE CASCADE ON DELETE CASCADE,
    service_id      INTEGER                                                                 NOT NULL
        CONSTRAINT host_services_services_id_fk
            REFERENCES services
            ON UPDATE CASCADE ON DELETE CASCADE,
    active          INTEGER      DEFAULT 1                                                  NOT NULL,
    schedule_number INTEGER      DEFAULT 3                                                  NOT NULL,
    schedule_unit   VARCHAR(255) DEFAULT 'm'::CHARACTER VARYING                             NOT NULL,
    last_check      TIMESTAMP    DEFAULT '0001-01-01 00:00:01'::TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    created_at      TIMESTAMP                                                               NOT NULL,
    updated_at      TIMESTAMP                                                               NOT NULL,
    status          VARCHAR(255) DEFAULT 'pending'::CHARACTER VARYING                       NOT NULL,
    last_message    VARCHAR(255) DEFAULT ''::CHARACTER VARYING                              NOT NULL
);

CREATE TABLE events
(
    id              SERIAL
        PRIMARY KEY,
    event_type      VARCHAR(255) NOT NULL,
    host_service_id INTEGER      NOT NULL,
    host_id         INTEGER      NOT NULL,
    service_name    VARCHAR(255) NOT NULL,
    host_name       VARCHAR(255) NOT NULL,
    message         VARCHAR(512) NOT NULL,
    created_at      TIMESTAMP    NOT NULL,
    updated_at      TIMESTAMP    NOT NULL
);

