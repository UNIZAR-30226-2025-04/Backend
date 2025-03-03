 cmd
│   └── main.go
├── config
│   ├── postgres.go
│   ├── redis.go
│   └── swagger
│       ├── docs.go
│       ├── swagger.json
│       └── swagger.yaml
├── controllers
│   ├── friends.go
│   ├── inbox.go
│   ├── lobby_controller.go
│   ├── lobby_controller_test.go
│   ├── lobby.go
│   └── user.go
├── go.mod
├── go.sum
├── letras-img.png
├── middleware
│   └── auth.go
├── models
│   ├── game_profile.go
│   ├── lobby.go
│   ├── postgres
│   ├── redis
│   ├── TODO-PONER-EL-RESTO-QUE-NO-TENEMOS
│   ├── user.go
│   └── users.go
├── postgre_sql
│   ├── create.sql
│   ├── delete.sql
│   ├── populate.sql
│   └── testing_csv_files
│       ├── friendship_requests.csv
│       ├── friendships.csv
│       ├── game_invitations.csv
│       ├── game_lobbies.csv
│       ├── game_profiles.csv
│       ├── in_game_players.csv
│       └── users.csv
├── README.md
├── redis
│   ├── dump.rdb
│   ├── init.go
│   ├── redis.go
│   ├── redis_interface.go
│   └── redis_test.go
├── routes
│   ├── api_integration_test.go
│   └── routes.go
├── services
│   ├── redis
│   │   ├── dump.rdb
│   │   ├── init.go
│   │   ├── redis.go
│   │   ├── redis_interface.go
│   │   └── redis_test.go
│   └── sync
│       ├── sync_manager.go
│       └── sync_test.go
├── utils
│   ├── logger.go
│   └── utils.go
└── views
    └── json_structs.txt

