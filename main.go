package main

import (
	"fmt"
	"io"
	"net/http"

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

var baseApiUrl string = "http://localhost:3000" // Substituir pelo domínio caso o deploy seja realizado

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/generateSong", func(w http.ResponseWriter, r *http.Request) {

		client := &http.Client{}
		req, err := http.NewRequest(r.Method, baseApiUrl+"/api/custom_generate", r.Body)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		defer resp.Body.Close()

		// Só exibindo corpo da resposta para verificar se a geração foi bem sucedida
		body, err := io.ReadAll(resp.Body)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Println(string(body))

		_, err = io.Copy(w, resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}).Methods("POST")

	http.ListenAndServe(":8080", r)
}
