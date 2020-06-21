CREATE TABLE IF NOT EXISTS users (
    id integer primary key auto_increment,
    email varchar(255) NOT NULL UNIQUE,
    password blob,
    firstname varchar(255),
    surname varchar(255),
    activation_token varchar(255),
    activated bool
);
