CREATE TABLE IF NOT EXISTS games (
    user_identifier text,
    word text,
    guess_count int,
    seconds int, 
    timestamp int
);

CREATE INDEX user_identifier_idx ON games(user_identifier);
