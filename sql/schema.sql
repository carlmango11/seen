create table seen.jobs
(
    id binary(16) not null
        primary key,
    name varchar(20),
    status_id int null,
    ip varchar(16) null,
    error varchar(20)
);

create table seen.status
(
    id int auto_increment
        primary key,
    name varchar(20) null
);

insert into seen.status (name) values ('incoming');
insert into seen.status (name) values ('normalised');
insert into seen.status (name) values ('prepped');
insert into seen.status (name) values ('annotated');
insert into seen.status (name) values ('complete');
insert into seen.status (name) values ('error');