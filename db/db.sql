DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;

CREATE EXTENSION IF NOT EXISTS citext;

CREATE UNLOGGED TABLE IF NOT EXISTS users (
    nickname    CITEXT UNIQUE PRIMARY KEY,
    email       CITEXT UNIQUE NOT NULL,
    fullname    TEXT NOT NULL,
    about       TEXT NOT NULL
);

CREATE UNLOGGED TABLE IF NOT EXISTS forum (
    slug        CITEXT NOT NULL PRIMARY KEY,
    title       TEXT NOT NULL,
    posts       BIGINT DEFAULT 0 NOT NULL,
    threads     BIGINT DEFAULT 0 NOT NULL,
    "user"      CITEXT NOT NULL REFERENCES users(nickname)
);

CREATE UNLOGGED TABLE thread
(
    id          SERIAL NOT NULL PRIMARY KEY,
    created     TIMESTAMPTZ,
    slug        CITEXT,
    message     TEXT NOT NULL,
    title       TEXT NOT NULL,
    votes       INTEGER DEFAULT 0 NOT NULL,
    author      CITEXT NOT NULL REFERENCES users(nickname),
    forum       CITEXT NOT NULL REFERENCES forums(slug)
);
CREATE OR REPLACE FUNCTION update_user_forum() RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO users_forum (nickname, Slug) VALUES (NEW.author, NEW.forum) on conflict do nothing;
    return NEW;
end
$$ LANGUAGE plpgsql;


CREATE UNLOGGED TABLE post
(
    id          SERIAL NOT NULL PRIMARY KEY,
    created     TIMESTAMPTZ NOT NULL,
    isEdited    BOOLEAN DEFAULT FALSE NOT NULL,
    message     TEXT NOT NULL,
    author      CITEXT NOT NULL REFERENCES users(nickname),
    thread      BIGINT NOT NULL REFERENCES thread(id),
    forum       CITEXT NOT NULL REFERENCES forum(slug),
    parent      BIGINT NOT NULL,
    path        int[]  DEFAULT ARRAY[] :: INT[]
);

CREATE OR REPLACE FUNCTION update_path() RETURNS TRIGGER AS
$$
DECLARE
    parent_path         BIGINT[];
    first_parent_thread INT;
BEGIN
    IF (NEW.parent IS NULL) THEN
        NEW.path := array_append(new.path, new.id);
    ELSE
        SELECT path FROM post WHERE id = new.parent INTO parent_path;
        SELECT thread FROM post WHERE id = parent_path[1] INTO first_parent_thread;
        IF NOT FOUND OR first_parent_thread != NEW.thread THEN
            RAISE EXCEPTION 'parent is from different thread' USING ERRCODE = '00409';
        end if;

        NEW.path := NEW.path || parent_path || new.id;
    end if;
    UPDATE forum SET Posts=Posts + 1 WHERE lower(forum.slug) = lower(new.forum);
    RETURN new;
end
$$ LANGUAGE plpgsql;

CREATE UNLOGGED TABLE vote
(
    nickname citext NOT NULL REFERENCES users (nickname),
    voice    INT,
    idThread INT REFERENCES thread (id),

    UNIQUE (nickname, idThread)
);


CREATE UNLOGGED TABLE users_forum
(
    nickname citext NOT NULL,
    Slug     citext NOT NULL,
    FOREIGN KEY (nickname) REFERENCES users (nickname),
    FOREIGN KEY (Slug) REFERENCES forum (Slug),
    UNIQUE (nickname, Slug)
);

CREATE OR REPLACE FUNCTION insert_votes() RETURNS TRIGGER AS
$$
BEGIN
    UPDATE thread SET votes=(votes+NEW.voice) WHERE id=NEW.idThread;
    return NEW;
end
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_votes() RETURNS TRIGGER AS
$$
BEGIN
    UPDATE thread SET votes=(votes+NEW.voice*2) WHERE id=NEW.idThread;
    return NEW;
end
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_threads_count() RETURNS TRIGGER AS
$$
BEGIN
    UPDATE forum SET Threads=(Threads+1) WHERE LOWER(slug)=LOWER(NEW.forum);
    return NEW;
end
$$ LANGUAGE plpgsql;

CREATE TRIGGER thread_insert_user_forum AFTER INSERT ON thread FOR EACH ROW
EXECUTE PROCEDURE update_user_forum();

CREATE TRIGGER post_insert_user_forum AFTER INSERT ON post FOR EACH ROW
EXECUTE PROCEDURE update_user_forum();

CREATE TRIGGER path_update_trigger BEFORE INSERT ON post FOR EACH ROW
EXECUTE PROCEDURE update_path();

CREATE TRIGGER add_vote BEFORE INSERT ON vote FOR EACH ROW
EXECUTE PROCEDURE insert_votes();

CREATE TRIGGER add_thread_to_forum BEFORE INSERT ON thread FOR EACH ROW
EXECUTE PROCEDURE update_threads_count();

CREATE TRIGGER edit_vote BEFORE UPDATE ON vote FOR EACH ROW
EXECUTE PROCEDURE update_votes();

CREATE INDEX post_first_parent_thread_index ON post ((post.path[1]), thread);
CREATE INDEX post_first_parent_id_index ON post ((post.path[1]), id);
CREATE INDEX post_first_parent_index ON post ((post.path[1]));
CREATE INDEX post_path_index ON post ((post.path));
CREATE INDEX post_thread_index ON post (thread);
CREATE INDEX post_thread_id_index ON post (thread, id);

CREATE INDEX forum_slug_lower_index ON forum (lower(forum.Slug));

CREATE INDEX users_nickname_lower_index ON users (lower(users.Nickname));
CREATE INDEX users_nickname_index ON users ((users.Nickname));
CREATE INDEX users_email_index ON users (lower(Email));

CREATE INDEX users_forum_forum_user_index ON users_forum (lower(users_forum.Slug), nickname);
CREATE INDEX users_forum_user_index ON users_forum (nickname);

CREATE INDEX thread_slug_lower_index ON thread (lower(slug));
CREATE INDEX thread_slug_index ON thread (slug);
CREATE INDEX thread_slug_id_index ON thread (lower(slug), id);
CREATE INDEX thread_forum_lower_index ON thread (lower(forum));
CREATE INDEX thread_id_forum_index ON thread (id, forum);
CREATE INDEX thread_created_index ON thread (created);

CREATE INDEX vote_nickname ON vote (lower(nickname), idThread, voice);

CREATE INDEX post_path_id_index ON post (id, (post.path));
CREATE INDEX post_thread_path_id_index ON post (thread, (post.parent), id);

CREATE INDEX users_forum_forum_index ON users_forum ((users_forum.Slug));