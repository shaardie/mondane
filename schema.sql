CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(255) NOT NULL UNIQUE,
    password BLOB,
    firstname VARCHAR(255),
    surname VARCHAR(255),
    activation_token VARCHAR(255),
    activated BOOL
);

CREATE TABLE IF NOT EXISTS http_checks (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    user_id INTEGER NOT NULL,
    url VARCHAR(255) NOT NULL,
    FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS http_results (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    timestamp DATETIME NOT NULL,
    check_id INTEGER NOT NULL,
    success BOOL NOT NULL,
    status_code INTEGER NOT NULL,
    duration BIGINT NOT NULL,
    error VARCHAR(255) NOT NULL,
    FOREIGN KEY (check_id)
        REFERENCES http_checks (id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS alerts (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    user_id INTEGER NOT NULL,
    check_id INTEGER NOT NULL,
    check_type VARCHAR(255) NOT NULL,
    send_mail BOOL NOT NULL,
    last_send DATETIME NOT NULL,
    send_period BIGINT NOT NULL,
    FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);
