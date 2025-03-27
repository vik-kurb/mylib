create table if not exists mylib.writers(
    id serial primary key,
    family_name text not null,
    first_name text,
    birth_year integer,
    death_year integer
);

create table if not exists mylib.books(
    id serial primary key,
    name text not null,
    publish_year integer
);

create table if not exists mylib.book_writers(
    book_id integer references mylib.books(id),
    writer_id integer references mylib.writers(id)
);
