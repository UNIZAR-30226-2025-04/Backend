-- Limpiamos las tablas en orden inverso para evitar problemas de FK
TRUNCATE game_invitations, in_game_players, game_lobbies, friendship_requests, friendships, users, game_profiles CASCADE;

-- 1. Primero game_profiles porque users lo referencia
\copy game_profiles(username, user_stats, is_in_a_game) FROM './postgre_sql/testing_csv_files/game_profiles.csv' WITH (FORMAT CSV, HEADER, DELIMITER ',');

-- 2. Luego users
\copy users FROM './postgre_sql/testing_csv_files/users.csv' CSV HEADER;

-- 3. Friendships
\copy friendships FROM './postgre_sql/testing_csv_files/friendships.csv' CSV HEADER;

-- 4. Friendship requests
\copy friendship_requests FROM './postgre_sql/testing_csv_files/friendship_requests.csv' CSV HEADER;

-- 5. Game lobbies
\copy game_lobbies FROM './postgre_sql/testing_csv_files/game_lobbies.csv' CSV HEADER;

-- 6. In game players
\copy in_game_players FROM './postgre_sql/testing_csv_files/in_game_players.csv' CSV HEADER;

-- 7. Game invitations
\copy game_invitations FROM './postgre_sql/testing_csv_files/game_invitations.csv' CSV HEADER;
