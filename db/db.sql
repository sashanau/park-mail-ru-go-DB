DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;
CREATE EXTENSION IF NOT EXISTS CITEXT;

CREATE UNLOGGED TABLE users
(
    about    text,
    email    CITEXT UNIQUE,
    fullName text NOT NULL,
    nickname CITEXT COLLATE "ucs_basic" PRIMARY KEY
);

CREATE UNLOGGED TABLE forum
(
    "user"  CITEXT NOT NULL REFERENCES users(nickname),
    posts   BIGINT DEFAULT 0,
    slug    CITEXT PRIMARY KEY,
    threads INT    DEFAULT 0,
    title   text
);

CREATE UNLOGGED TABLE thread
(
    author  CITEXT             REFERENCES users (nickname),
    created timestamp          WITH TIME ZONE DEFAULT now(),
    forum   CITEXT             REFERENCES forum (slug),
    id      SERIAL             PRIMARY KEY,
    message text               NOT NULL,
    slug    CITEXT             UNIQUE,
    title   text               NOT NULL,
    votes   INT                DEFAULT 0
);

CREATE UNLOGGED TABLE post
(
    author   CITEXT NOT NULL REFERENCES users (nickname),
    created  timestamp with time zone default now(),
    forum    CITEXT REFERENCES forum (slug),
    id       BIGSERIAL PRIMARY KEY,
    isEdited BOOLEAN                  DEFAULT FALSE,
    message  text   NOT NULL,
    parent   BIGINT                   DEFAULT 0 REFERENCES post (id),
    thread   INT REFERENCES thread (id),
    path     BIGINT[]                 default array []::INTEGER[]
);

CREATE UNLOGGED TABLE vote
(
    nickname CITEXT NOT NULL REFERENCES users (nickname),
    voice    INT,
    idThread INT REFERENCES thread (id),
    UNIQUE (nickname, idThread)
);


CREATE UNLOGGED TABLE users_forum
(
    nickname CITEXT COLLATE "ucs_basic" NOT NULL REFERENCES users (nickname),
    Slug     CITEXT NOT NULL REFERENCES forum (Slug),
    UNIQUE (nickname, Slug)
);

CREATE OR REPLACE FUNCTION update_user_forum() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO users_forum (nickname, Slug) VALUES (NEW.author, NEW.forum) on conflict do nothing;
    return NEW;
end
$$ LANGUAGE plpgsql;

CREATE TRIGGER thread_insert_user_forum AFTER INSERT ON thread FOR EACH ROW EXECUTE PROCEDURE update_user_forum();
CREATE TRIGGER post_insert_user_forum AFTER INSERT ON post FOR EACH ROW EXECUTE PROCEDURE update_user_forum();


CREATE OR REPLACE FUNCTION update_path() RETURNS TRIGGER AS $$
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

CREATE TRIGGER path_update_trigger BEFORE INSERT ON post FOR EACH ROW EXECUTE PROCEDURE update_path();

CREATE OR REPLACE FUNCTION insert_votes() RETURNS TRIGGER AS $$
BEGIN
    UPDATE thread SET votes=(votes+NEW.voice) WHERE id=NEW.idThread;
    return NEW;
end
$$ LANGUAGE plpgsql;

CREATE TRIGGER add_vote BEFORE INSERT ON vote FOR EACH ROW EXECUTE PROCEDURE insert_votes();

CREATE OR REPLACE FUNCTION update_votes() RETURNS TRIGGER AS $$
BEGIN
    UPDATE thread SET votes=(votes+NEW.voice*2) WHERE id=NEW.idThread;
    return NEW;
end
$$ LANGUAGE plpgsql;

CREATE TRIGGER edit_vote BEFORE UPDATE ON vote FOR EACH ROW EXECUTE PROCEDURE update_votes();

CREATE OR REPLACE FUNCTION update_threads_count() RETURNS TRIGGER AS $$
BEGIN
    UPDATE forum SET Threads=(Threads+1) WHERE LOWER(slug)=LOWER(NEW.forum);
    return NEW;
end
$$ LANGUAGE plpgsql;

CREATE TRIGGER add_thread_to_forum BEFORE INSERT ON thread FOR EACH ROW EXECUTE PROCEDURE update_threads_count();



CREATE INDEX post_path_thread ON post ((post.path[1]), thread);
CREATE INDEX post_path1_id ON post ((post.path[1]), id);
CREATE INDEX post_path1 ON post ((post.path[1]));
CREATE INDEX post_path ON post ((post.path));
CREATE INDEX post_thread ON post (thread);
CREATE INDEX post_thread_id ON post (thread, id);
CREATE INDEX forum_slug_lower ON forum (lower(forum.Slug));
CREATE INDEX users_nickname_lower ON users (lower(users.Nickname));
CREATE INDEX users_nickname ON users ((users.Nickname));
CREATE INDEX users_email ON users (lower(Email));
CREATE INDEX users_forum_slug_nickname ON users_forum (lower(users_forum.Slug), nickname);
CREATE INDEX users_forum_nickname ON users_forum (nickname);
CREATE INDEX thread_slug_lower ON thread (lower(slug));
CREATE INDEX thread_slug ON thread (slug);
CREATE INDEX thread_slug_lower_id ON thread (lower(slug), id);
CREATE INDEX thread_forum_lower ON thread (lower(forum));
CREATE INDEX thread_id_forum ON thread (id, forum);
CREATE INDEX thread_created ON thread (created);
CREATE INDEX vote_nickname_idThread_voice ON vote (lower(nickname), idThread, voice);
CREATE INDEX post_id_path ON post (id, (post.path));
CREATE INDEX post_thread_parent_id ON post (thread, (post.parent), id);
CREATE INDEX users_forum_slug ON users_forum ((users_forum.Slug));


-- explain SELECT nickname, fullname, about, email FROM forum_users WHERE forum = '123' AND nickname > '1234'
--                                   ORDER BY nickname
--                                       LIMIT 17;
-- OK
-- explain SELECT id, title, author, forum, message, votes, slug, created
--                                        FROM threads
--                                        WHERE forum = 'xvE3J8FuYwj9r' AND created <= 'Sun Jun 26 2022 14:41:14 GMT+0000'
--                                        ORDER BY created
--                                        LIMIT 17::TEXT::INTEGER;
-- OK

-- explain SELECT id, title, author, forum, message, votes, slug, created FROM threads WHERE slug = '214';
-- OK

-- explain SELECT slug, title, "user", posts, threads FROM forums WHERE slug = '3Q6wNC4CyYc9k';
-- explain SELECT nickname, fullname, about, email FROM users WHERE nickname = 'tuam.T1ffNf4F885tPM';
--  SELECT id, parent, author, message, isedited, forum, thread, created FROM posts WHERE id = 764093;

-- explain SELECT id, parent, author, message, isEdited, forum, thread, created
--                                         FROM posts
--                                         WHERE thread = 0
--                                         ORDER BY path
--                                         LIMIT 17::TEXT::INTEGER;
-- -- OK
-- --
-- explain SELECT id, parent, author, message, isEdited, forum, thread, created FROM posts
--                                         WHERE path[1] IN (SELECT id FROM posts WHERE thread = 0 AND parent = 0 AND id < (SELECT path[1] FROM posts WHERE id = 0)
--                                                           ORDER BY id DESC LIMIT 17) ORDER BY path ASC;
-- OK
-- explain SELECT id, parent, author, message, isEdited, forum, thread, created FROM posts
--                                         WHERE path[1] IN (SELECT id FROM posts WHERE thread = 1 AND parent = 0 ORDER BY id DESC LIMIT 17)
--                                         ORDER BY path ASC;
-- OK

-- explain SELECT id, parent, author, message, isEdited, forum, thread, created
--                                        FROM posts
--                                        WHERE thread = 1
--                                        ORDER BY id
--                                            LIMIT 17::TEXT::INTEGER;
-- OK

-- explain SELECT * FROM users WHERE nickname = '4124';
-- OK

-- explain SELECT id, parent, author, message, isedited, forum, thread, created FROM posts WHERE id = 1;
-- OK
--
-- explain SELECT title, "user", slug, posts, threads FROM forums WHERE slug = '123';
-- OK

-- explain SELECT id, title, author, forum, message, votes, slug, created FROM threads WHERE id = 1;
-- OK


-- explain SELECT * FROM threads WHERE slug = '2141';
--
-- explain SELECT id, parent, author, message, isEdited, forum, thread, created
--                                        FROM posts
--                                        WHERE thread = 0 AND id > 0
--                                        ORDER BY id
--                                            LIMIT 17::TEXT::INTEGER;