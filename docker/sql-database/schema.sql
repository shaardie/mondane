create table if not exists users (
    id integer primary key auto_increment,
    email varchar(255) NOT NULL UNIQUE,
    password blob,
    firstname varchar(255),
    surname varchar(255),
    activation_token varchar(255),
    activated bool
);

create table if not exists http_checks (
    id integer not null primary key auto_increment,
    user_id integer not null,
    url varchar(255) not null,
    foreign key (user_id) references users(id) on delete cascade
);

create table if not exists http_check_results (
    id integer not null primary key auto_increment,
    timestamp datetime not null,
    check_id integer not null,
    success bool not null,
    foreign key (check_id) references http_checks(id) on delete cascade
);

create table if not exists alerts (
    id integer not null primary key auto_increment,
    user_id integer not null,
    check_id integer not null,
    check_type varchar(255) not null,
    send_mail bool not null,
    last_send datetime not null,
    send_period bigint not null,
    foreign key (user_id) references users(id) on delete cascade
);
