-- Game profiles table
CREATE TABLE game_profiles (
    -- Primary keys
    username VARCHAR(50) PRIMARY KEY NOT NULL,

    -- Attributes
    user_stats JSONB DEFAULT '{}', -- We choose JSONB as it's a flexible type that can store complex data
    user_icon INTEGER DEFAULT 0,
    is_in_a_game BOOLEAN DEFAULT FALSE -- TODO: Check if this is ACTUALLY needed
);

-- Users table
CREATE TABLE users (
    -- Primary keys
    email VARCHAR(100) PRIMARY KEY NOT NULL,

    -- Foreign keys
    username VARCHAR(50) NOT NULL REFERENCES game_profiles(username),

    -- Attributes
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100),
    member_since TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Friends table
CREATE TABLE friendships (
    -- Primary keys
    username1 VARCHAR(50) NOT NULL REFERENCES game_profiles(username),
    username2 VARCHAR(50) NOT NULL REFERENCES game_profiles(username),
    PRIMARY KEY (username1, username2),

    -- Checks
    CHECK (username1 <> username2)
);

-- Friendship requests table
CREATE TABLE friendship_requests (
    -- Primary keys
    username1 VARCHAR(50) NOT NULL REFERENCES game_profiles(username),
    username2 VARCHAR(50) NOT NULL REFERENCES game_profiles(username),
    PRIMARY KEY (username1, username2),

    -- Attributes
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Checks
    CHECK (username1 <> username2)
);

-- Game lobbies table
CREATE TABLE game_lobbies (
    -- Primary keys
    id VARCHAR(50) PRIMARY KEY NOT NULL,
    
    -- Attributes
    creator_username VARCHAR(50) REFERENCES game_profiles(username),
    number_of_rounds INTEGER DEFAULT 0,
    total_points INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- In game players table
CREATE TABLE in_game_players (
    -- Primary keys
    lobby_id VARCHAR(50) REFERENCES game_lobbies(id),
    username VARCHAR(50) REFERENCES game_profiles(username),
    PRIMARY KEY (lobby_id, username),

    -- Attributes
    players_money INTEGER DEFAULT 0,
    most_played_hand JSONB DEFAULT '{}',
    winner BOOLEAN DEFAULT FALSE -- Since there might be two or more winners. (ties are possible)
);

-- Game invitations table
CREATE TABLE game_invitations (
    -- Primary keys
    lobby_id VARCHAR(50) REFERENCES game_lobbies(id),
    invited_username VARCHAR(50) REFERENCES game_profiles(username),
    PRIMARY KEY (lobby_id, invited_username),

    -- Attributes
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for better performance
CREATE INDEX idx_friendships_username2 ON friendships(username2);
CREATE INDEX idx_game_lobbies_creator ON game_lobbies(creator_username);
CREATE INDEX idx_in_game_players_username ON in_game_players(username);
