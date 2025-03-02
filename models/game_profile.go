package models

type Game_profile struct {
	Username     string `json:"username"`
	Is_in_a_game bool   `json:"is_in_a_game"`
}

/**solicita_amistad and friend_of are here
because it is a reflexive relationship
(game_profile - game_profile)
*
*/

type Solicita_amistad struct {
	Fecha_creacion string `json:"fecha_creacion"`
}
