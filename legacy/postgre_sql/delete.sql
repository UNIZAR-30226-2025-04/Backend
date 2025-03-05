-- Drop all the tables
DROP TABLE game_invitations;
DROP TABLE in_game_players;
DROP TABLE game_lobbies;
DROP TABLE friendships;
DROP TABLE friendship_requests;
DROP TABLE users;
DROP TABLE game_profiles;

-- Drop all the indexes
DROP INDEX idx_friendships_username2;
DROP INDEX idx_game_lobbies_creator;
DROP INDEX idx_in_game_players_username;