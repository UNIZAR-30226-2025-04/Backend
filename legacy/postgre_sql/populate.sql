-- Clean tables in reverse order to avoid FK issues
TRUNCATE game_invitations, in_game_players, game_lobbies, friendship_requests, friendships, users, game_profiles CASCADE;

-- 1. First game_profiles because users references it
\copy game_profiles(username, user_stats, user_icon, is_in_a_game) FROM './postgre_sql/testing_csv_files/game_profiles.csv' WITH (FORMAT CSV, HEADER, DELIMITER ',');

-- 2. Then users
\copy users(email, username, password_hash, full_name, member_since) FROM './postgre_sql/testing_csv_files/users.csv' CSV HEADER;

-- 3. Friendships
\copy friendships(username1, username2) FROM './postgre_sql/testing_csv_files/friendships.csv' CSV HEADER;

-- 4. Friendship requests
\copy friendship_requests(username1, username2, created_at) FROM './postgre_sql/testing_csv_files/friendship_requests.csv' CSV HEADER;

-- 5. Game lobbies
\copy game_lobbies(id, creator_username, number_of_rounds, total_points, created_at) FROM './postgre_sql/testing_csv_files/game_lobbies.csv' CSV HEADER;

-- 6. In game players
\copy in_game_players(lobby_id, username, players_money, most_played_hand, winner) FROM './postgre_sql/testing_csv_files/in_game_players.csv' CSV HEADER;

-- 7. Game invitations
\copy game_invitations(lobby_id, invited_username, created_at) FROM './postgre_sql/testing_csv_files/game_invitations.csv' CSV HEADER;
