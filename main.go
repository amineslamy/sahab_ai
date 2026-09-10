package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	Neo4jURI string `json:"neo4j_uri"`
	RedisURL string `json:"redis_url"`
	TursoURL string `json:"turso_url"`
}

type ProcessRequest struct {
	Text string `json:"text"`
}

func main() {
	_ = godotenv.Load()

	// API کانفیگ
	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		cfg := Config{
			BaseURL:  os.Getenv("OPENAI_BASE_URL"),
			Model:    os.Getenv("LLM_MODEL"),
			Neo4jURI: os.Getenv("NEO4J_URI"),
			RedisURL: os.Getenv("REDIS_URL"),
			TursoURL: os.Getenv("TURSO_DATABASE_URL"),
		}
		json.NewEncoder(w).Encode(cfg)
	})

	// API جدید برای پردازش متن با اولاما
	http.HandleFunc("/api/process", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, "فقط روش POST مجاز است", http.StatusMethodNotAllowed)
			return
		}

		var req ProcessRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		// تست ارتباط با اولاما
		ollamaURL := os.Getenv("OPENAI_BASE_URL") + "/chat/completions"
		modelName := os.Getenv("LLM_MODEL")

		payload := map[string]interface{}{
			"model": modelName,
			"messages": []map[string]string{
				{"role": "user", "content": "این متن را به صورت خلاصه تایید کن: " + req.Text},
			},
		}
		jsonPayload, _ := json.Marshal(payload)

		resp, err := http.Post(ollamaURL, "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			http.Error(w, "خطا در ارتباط با اولاما: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		// ارسال پاسخ تست به فرانت‌اند به همراه داده تعاملی D3
		w.Write(json.RawMessage(fmt.Sprintf(`{
			"status": "success",
			"ollama_raw": %s,
			"nodes": [{"id": "متن ورودی", "group": 1}, {"id": "اولاما (Qwen)", "group": 2}],
			"links": [{"source": "متن ورودی", "target": "اولاما (Qwen)", "value": 1}]
		}`, string(body))))
	})

	// سرو کردن ui
	fs := http.FileServer(http.Dir("./ui"))
	http.Handle("/", fs)

	fmt.Println("==================================================")
	fmt.Println("🚀 وب سرور سحاب AI آماده است: http://localhost:8081")
	fmt.Println("==================================================")

	log.Fatal(http.ListenAndServe(":8081", nil))
}
