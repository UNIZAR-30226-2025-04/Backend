-- Users table
CREATE TABLE users (
    -- Primary keys
    email VARCHAR(100) PRIMARY KEY NOT NULL,

    -- Foreign keys
    username VARCHAR(50) NOT NULL REFERENCES game_profiles(username),

    -- Attributes
    email_verified BOOLEAN DEFAULT FALSE,
    id VARCHAR(50) UNIQUE, -- As of right now we leave it like this. TODO: Define a better type for the id
    full_name VARCHAR(100),
    member_since TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    settings JSONB DEFAULT '{}' -- We choose JSONB as it's a flexible type that can store complex data
);

-- Game profiles table
CREATE TABLE game_profiles (
    -- Primary keys
    username VARCHAR(50) PRIMARY KEY NOT NULL,

    -- Attributes
    user_stats JSONB DEFAULT '{}', -- We choose JSONB as it's a flexible type that can store complex data
    is_in_a_game BOOLEAN DEFAULT FALSE, -- TODO: Check if this is ACTUALLY needed
);

-- Friends table
CREATE TABLE friendships (
    -- Primary keys
    PRIMARY KEY NOT NULL (username1, username2),

    -- Foreign keys, username1 and username2 cannot be the same
    username1 VARCHAR(50) NOT NULL REFERENCES game_profiles(username),
    username2 VARCHAR(50) NOT NULL REFERENCES game_profiles(username),

    -- Checks
    CHECK (username1 <> username2)
);

-- Friendship requests table
CREATE TABLE friendship_requests (
    -- Primary keys
    PRIMARY KEY NOT NULL (username1, username2),

    -- Foreign keys, username1 and username2 cannot be the same
    username1 VARCHAR(50) NOT NULL REFERENCES game_profiles(username),
    username2 VARCHAR(50) NOT NULL REFERENCES game_profiles(username),

    -- Attributes
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP

    -- Checks
    CHECK (username1 <> username2)
);

-- Game lobbies table
CREATE TABLE game_lobbies (
    -- Primary keys
    id VARCHAR(50) PRIMARY KEY NOT NULL,

    -- Attributes
    number_of_rounds INTEGER DEFAULT 0,
    total_points INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- In game players table
CREATE TABLE in_game_players (
    -- Primary keys
    PRIMARY KEY (lobby_id, username),

    -- Foreign keys
    lobby_id VARCHAR(50) REFERENCES game_lobbies(lobby_id),
    username VARCHAR(50) REFERENCES users(username),

    -- Attributes
    players_money INTEGER DEFAULT 0,
    current_deck JSONB DEFAULT '{}',
    current_modifiers JSONB DEFAULT '{}',
    current_wildcards JSONB DEFAULT '{}',
    winner BOOLEAN DEFAULT FALSE -- Since there might be two or more winners. (ties are possible)
);

-- Game invitations table
CREATE TABLE game_invitations (
    -- Primary keys
    PRIMARY KEY (lobby_id, invited_username),

    -- Foreign keys
    lobby_id VARCHAR(50) REFERENCES game_lobbies(lobby_id),
    invited_username VARCHAR(50) REFERENCES users(username),

    -- Attributes
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
);

-- Indexes for better performance
CREATE INDEX idx_friendships_username2 ON friendships(username2);
CREATE INDEX idx_game_lobbies_creator ON game_lobbies(creator_username);
CREATE INDEX idx_in_game_players_username ON in_game_players(username);
