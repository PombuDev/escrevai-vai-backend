package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Player struct {
	Nick string `json:"username"`
}

type Lobby struct {
	Id          string   `json:"id"`
	Players     []Player `json:"players"`
	PromptTitle string   `json:"prompttitle"`
	PromptGenre string   `json:"promptgenre"`
	PromptText  string   `json:"prompttext"`
	ResultSong  string   `json:"resultsong"`
}

var baseApiUrl string = "http://localhost:3000" // Substituir pelo domínio caso o deploy seja realizado
var lobbies map[string]Lobby = make(map[string]Lobby)

func generateId() string {
	uuid := uuid.New()
	return uuid.String()
}

// TODO: Fazer integração com firebase

func main() {

	router := gin.Default()

	// Criar lobby
	router.POST("/lobby", func(c *gin.Context) {
		var player Player

		if err := c.ShouldBindJSON(&player); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		players := []Player{player}

		id := generateId()
		newLobby := Lobby{Id: id, Players: players}
		lobbies[id] = newLobby

		c.JSON(http.StatusCreated, gin.H{"lobbyid": id})
	})

	// Entrar em lobby existente
	router.GET("/lobby/:id", func(c *gin.Context) {
		id := c.Param("id")

		if len(lobbies[id].Players) == 4 {
			c.JSON(http.StatusForbidden, gin.H{"message": "servidor cheio"})
			return
		}

		var player Player

		if err := c.ShouldBindJSON(&player); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		lobby, exists := lobbies[id]

		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"message": "lobby informado não existe"})
			return
		}

		lobby.Players = append(lobby.Players, player)
		lobbies[id] = lobby

		c.JSON(http.StatusFound, gin.H{"lobbyfounded": lobby})
	})

	// Gerar música
	router.POST("/lobby/:id/song", func(c *gin.Context) {
		client := &http.Client{}

		req, err := http.NewRequest("POST", baseApiUrl+"/api/custom_generate", c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		// TODO: descobrir como fazer para recuperar as infos da música gerada, usando solução provisória

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		fmt.Println(string(body))

		c.JSON(http.StatusCreated, gin.H{"data": string(body)})

	})

	router.Run(":8080")
}
