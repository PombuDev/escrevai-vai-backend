package main

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CustomSong struct {
	Id        string `json:"id"`
	Title     string `json:"title"`
	ImageURL  string `json:"image_url"`
	Lyric     string `json:"lyric"`
	AudioURL  string `json:"audio_url"`
	VideoURL  string `json:"video_url"`
	CreatedAt string `json:"created_at"`
	ModelName string `json:"model_name"`
	Status    string `json:"status"`
	GPTDesc   string `json:"gpt_description_prompt"`
	Prompt    string `json:"prompt"`
	Type      string `json:"type"`
	Tags      string `json:"tags"`
}

type Player struct {
	// Image image?
	Username string `json:"username"`
}

type Lobby struct {
	// Players     []string `json:"players"`
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
// TODO: Considerar refazer com gin

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/generateSong", func(w http.ResponseWriter, r *http.Request) {
		client := &http.Client{}
		req, err := http.NewRequest("POST", baseApiUrl+"/api/custom_generate", r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var song CustomSong
		json.NewDecoder(resp.Body).Decode(&song)

		defer resp.Body.Close()

		//TODO: Descobrir como fazer para devolvar a resposta, já que a song tá completamente vazia

		json.NewEncoder(w).Encode(song)
		// // Só exibindo corpo da resposta para verificar se a geração foi bem sucedida
		// body, err := io.ReadAll(resp.Body)
		// if err != nil {
		// 	http.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }
		// fmt.Println(string(body))
		// _, err = io.Copy(w, resp.Body)
		// if err != nil {
		// 	http.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }
	}).Methods("GET")

	r.HandleFunc("/createLobby", func(w http.ResponseWriter, r *http.Request) {
		var player Player

		// TODO: Descobrir porque o nome do player não aparece na criação do lobby, mandando um vazio

		json.NewDecoder(r.Body).Decode(&player)

		players := []Player{player}

		id := generateId()
		newLobby := Lobby{Id: id, Players: players}
		lobbies[id] = newLobby

		json.NewEncoder(w).Encode(newLobby)
	}).Methods("POST")

	r.HandleFunc("/joinLobby/{id}", func(w http.ResponseWriter, r *http.Request) {
		var player Player

		json.NewDecoder(r.Body).Decode(&player)

		vars := mux.Vars(r)

		lobby := lobbies[vars["id"]]
		lobby.Players = append(lobby.Players, player)
		lobbies[vars["id"]] = lobby
		json.NewEncoder(w).Encode(lobby)
	}).Methods("GET")

	r.HandleFunc("/gameId={id}", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Fazer a lógica de quando se inicia um jogo
		// Penso que tem que ser algo com a função randomica, sorteando quem vai fazer cada um. Talvez tivesse que colocar um campo bool no player pra verificar se ele já jogou

	}).Methods("GET")

	r.HandleFunc("/gameId={id}&setGenre", func(w http.ResponseWriter, r *http.Request) {
		var genre struct {
			Genre string `json:"genre"`
		}

		json.NewDecoder(r.Body).Decode(&genre)

		vars := mux.Vars(r)

		lobby := lobbies[vars["id"]]
		lobby.PromptGenre = genre.Genre
		lobbies[vars["id"]] = lobby

		json.NewEncoder(w).Encode(lobby)

	}).Methods("POST")

	http.ListenAndServe(":8080", r)
}
