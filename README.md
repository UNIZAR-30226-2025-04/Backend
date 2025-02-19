
# Nogler's backend

This will be the backend for "Nogler", a multiplayer game based on "balatro" by LocalThunk, focused on creating a entertaining experience for players, in a multiplayer environment.




## Authors

- [@JSerranom04](https://github.com/JSerranom04)

- [@Yago-Torres](https://github.com/Yago-Torres)

- [@nicolas-pueyo](https://github.com/nicolas-pueyo)

## Dependencies
go version 1.24
gin-gonic (https://github.com/gin-gonic/gin) version 1.10

## Deployment

To deploy this project run

```bash
  npm run deploy
```
_add here deployment details_

![Logo](letras-img.png)

_Nogler logo here_

## Usage/Examples

```javascript
import Component from 'my-project'

function App() {
  return <Component />
}
```
## Design decisions

### Postgre SQL database

# Design Decisions and Justifications:

1. Email as Primary Key (users table):
   - Emails are unique and globally identifiable
   - More reliable for user identification than usernames
   - Supports standard authentication practices
   - Username stored as foreign key to game_profiles

2. Game Profiles Separation:
   - Separates gaming-specific data from user account data
   - Username as primary key for gaming interactions
   - Allows for gaming-specific boolean flags (is_in_a_game, solicita_amistad, etc.)
   - Maintains clean separation of concerns

3. JSONB for Flexible Fields:
   - settings: Allows for flexible user configuration storage
   - user_stats: Game statistics can evolve without schema changes
   - baraja_actual/modificadores/comodines_actuales: Game state needs flexible structure
   - Better performance than regular JSON in PostgreSQL

4. Friendship System Design:
   - Split into friendships and friendship_requests tables
   - Bidirectional relationship using username1 and username2
   - CHECK constraint prevents self-friendships
   - References game_profiles instead of users table

5. Game Lobby Structure:
   - Uses string-based IDs for flexibility
   - Tracks rounds and points
   - Supports multiple concurrent games

6. In-Game Players Design:
   - Composite primary key (lobby_id, username)
   - Stores current game state (money, deck, modifiers)
   - Winner flag for game resolution (although it can be calculated as of right now it stands like this until further discussion in order to avoid extra calculations)
   - UNIQUE constraint prevents duplicate players in same lobby

7. Game Invitations Design:
   - Simplified structure with just lobby and invited user
   - Composite primary key prevents duplicate invitations
   - Timestamp tracking for invitation management

8. Indexing:
   - Indexes on frequently queried columns
   - Focus on foreign key fields
   - Optimized for common join operations



_reference api here, code or postman package or..._
