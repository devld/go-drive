-- DDL

CREATE TABLE drives
(
    name    VARCHAR
        PRIMARY KEY,
    type    VARCHAR NOT NULL,
    config  VARCHAR NOT NULL,
    enabled INTEGER
);

CREATE TABLE path_mount
(
    path     VARCHAR,
    name     VARCHAR,
    mount_at VARCHAR NOT NULL,
    PRIMARY KEY (path, name)
);

CREATE TABLE groups
(
    name VARCHAR
        PRIMARY KEY
);

CREATE TABLE path_permissions
(
    path       VARCHAR,
    subject    VARCHAR NOT NULL,
    permission INTEGER NOT NULL,
    policy     INTEGER NOT NULL,
    depth      INTEGER NOT NULL,
    PRIMARY KEY (path, subject)
);

CREATE TABLE user_groups
(
    group_name VARCHAR,
    username   VARCHAR,
    PRIMARY KEY (group_name, username)
);

CREATE TABLE users
(
    username VARCHAR
        PRIMARY KEY,
    password VARCHAR NOT NULL
);

CREATE TABLE drive_data
(
    drive      VARCHAR,
    data_key   VARCHAR,
    data_value VARCHAR,
    PRIMARY KEY (drive, data_key)
);

CREATE TABLE drive_cache
(
    drive       VARCHAR NOT NULL,
    path        VARCHAR NOT NULL,
    depth       INTEGER NOT NULL,
    type        INTEGER NOT NULL,
    cache_value TEXT    NOT NULL,
    expires_at  INTEGER NOT NULL,
    PRIMARY KEY (drive, path, depth, type)
);

-- Init data

INSERT INTO users(username, password)
VALUES ('admin', '$2y$10$Xqn8qV2D2KY2ceI5esM/JOiKTPKJFbkSzzuhce89BxygvCqnhyk3m');
-- 123456

INSERT INTO groups(name)
VALUES ('admin');

INSERT INTO user_groups(username, group_name)
VALUES ('admin', 'admin');

INSERT INTO path_permissions(path, subject, permission, policy, depth)
VALUES ('', 'ANY', 1, 1, 0);
INSERT INTO path_permissions(path, subject, permission, policy, depth)
VALUES ('', 'g:admin', 3, 1, 0);
