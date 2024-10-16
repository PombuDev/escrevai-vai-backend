package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Player struct {
	Nick string `json:"username"`
}

type Lobby struct {
	Id          string   `json:"id"`
	Players     []Player `json:"players"`
	PromptTitle string   `json:"prompttitle"`
	PromptGenre string   `json:"promptgenre"`
	ResultSong  []string `json:"resultsong"`
	Strophes    []string `json:"strophes"`
	CurrentTurn int      `json:"currentturn"`
}

type SongAPIResponse struct {
	ID                   string `json:"id"`
	Title                string `json:"title"`
	ImageURL             string `json:"image_url"`
	Lyric                string `json:"lyric"`
	AudioURL             string `json:"audio_url"`
	VideoURL             string `json:"video_url"`
	CreatedAt            string `json:"created_at"`
	ModelName            string `json:"model_name"`
	Status               string `json:"status"`
	GPTDescriptionPrompt string `json:"gpt_description_prompt"`
	Prompt               string `json:"prompt"`
	Type                 string `json:"type"`
	Tags                 string `json:"tags"`
}

var baseApiUrl string = "http://localhost:3000" // Substituir pelo domínio caso o deploy seja realizado
var lobbies = make(map[string]Lobby)

var wsConns = make(map[string]*websocket.Conn)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func generateId() string {
	uuid := uuid.New()
	return uuid.String()
}

func notifyPlayer(playername string, message string) error {
	conn, exists := wsConns[playername]
	if !exists {
		return fmt.Errorf("player %s não tá conectado", playername)
	}

	return conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func handleWebSocketMessage(playername string, message []byte) {
	fmt.Printf("recebeu mensagem de %s: %s\n", playername, message)
}

func generatePrompt(strophes []string) string {
	return strings.Join(strophes, "\n")
}

// TODO: Fazer integração com firebase

func main() {

	router := gin.Default()

	// Route para websockets
	router.GET("/ws/:playername", func(c *gin.Context) {
		playername := c.Param("playername")

		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "não conseguiu setar a atualização do websocket"})
			return
		}

		wsConns[playername] = ws
		defer ws.Close()

		for {
			_, message, err := ws.ReadMessage()
			if err != nil {
				delete(wsConns, playername)
				break
			}

			handleWebSocketMessage(playername, message)
		}
	})

	// Criar lobby
	router.POST("/lobby", func(c *gin.Context) {
		var player Player

		if err := c.ShouldBindJSON(&player); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		players := []Player{player}

		id := generateId()
		newLobby := Lobby{Id: id, Players: players, PromptTitle: "MusicTitle", CurrentTurn: 0}
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
		id := c.Param("id")

		lobby, exists := lobbies[id]
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "lobby não encontrado"})
			return
		}

		var playerInput struct {
			Playername string `json:"playername"`
			Content    string `json:"content"`
		}

		if err := c.ShouldBindJSON(&playerInput); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verificando se é o turno do jogador
		if lobby.Players[lobby.CurrentTurn].Nick != playerInput.Playername {
			c.JSON(http.StatusForbidden, gin.H{"error": "não é seu turno"})
			return
		}

		if lobby.CurrentTurn == 0 {
			lobby.PromptGenre = playerInput.Content
		} else {
			lobby.Strophes = append(lobby.Strophes, playerInput.Content)
		}

		fmt.Printf("Turno atual: %d\n", lobby.CurrentTurn)

		lobby.CurrentTurn++

		lobbies[id] = lobby

		if lobby.CurrentTurn > len(lobby.Players)-1 {

			requestBody := map[string]interface{}{
				"prompt":            generatePrompt(lobby.Strophes),
				"tags":              lobby.PromptGenre,
				"title":             lobby.PromptTitle,
				"make_instrumental": false,
				"wait_audio":        true,
			}

			jsonBody, err := json.Marshal(requestBody)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "falhou ao tentar \"marshalizar\" o corpo da requisição"})
				return
			}

			client := &http.Client{}

			req, err := http.NewRequest("POST", baseApiUrl+"/api/custom_generate", bytes.NewBuffer(jsonBody))
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

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
				return
			}

			fmt.Println("API Response Body:", string(body))

			var apiResponse []SongAPIResponse
			if err := json.Unmarshal(body, &apiResponse); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "falha ao decodificar a resposta"})
				return
			}

			var resultSongs []string

			for i, song := range apiResponse {
				response := fmt.Sprintf("Música %d\nAudio: %s\nTitle: %s\nImage: %s\nLyrics: %s\nGenre: %s\n", i, song.AudioURL, song.Title, song.ImageURL, song.Lyric, song.Tags)
				resultSongs = append(lobby.ResultSong, response)
			}

			lobby.ResultSong = append(lobby.ResultSong, resultSongs...)

			c.JSON(http.StatusCreated, gin.H{
				"message": "jogo finalizado e musicas criada",
				"song 1":  lobby.ResultSong[0],
				"song 2":  lobby.ResultSong[1],
			})

		} else {
			nextPlayer := lobby.Players[lobby.CurrentTurn].Nick
			notifyPlayer(nextPlayer, "seu turno de adicionar algo")
			c.JSON(http.StatusOK, gin.H{"message": "verso adicionado", "nextturn": nextPlayer})
		}
	})

	router.Run(":8080")
}
