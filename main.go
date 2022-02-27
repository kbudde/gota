package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"

	"net/http"
)

type Specification struct {
	BasePath      string `default:"/tmp/gota"`
	Port          int    `default:"8080"`
	FormField     string `default:"file"`
	MaxFileSizeMB int64  `default:"5"`
}

var config = Specification{}

func initConfig() error {
	return envconfig.Process("gota", &config)
}

func setupHttpServer() {
	http.HandleFunc("/upload", postHandler())
	http.HandleFunc("/download/", getHandler())
}

func main() {
	err := initConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to process env config")
	}

	err = os.Mkdir(config.BasePath, 0700)
	if err != nil && !errors.Is(err, os.ErrExist) {
		log.Fatal().Err(err).Msg("failed to create base path")
	}

	setupHttpServer()

	// start http server
	log.Info().Msg("Starting http server")
	log.Fatal().Err(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)).Msg("failed to start http server")

}

func getHandler() http.HandlerFunc {
	log := log.With().Str("handler", "getHandler").Logger()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			log.Warn().Msg("Method not allowed")
			return
		}
		log.Info().Str("path", r.URL.Path).Msg("GET request")
		http.StripPrefix("/download/", http.FileServer(http.Dir(config.BasePath))).ServeHTTP(w, r)
	}
}

func postHandler() http.HandlerFunc {
	log := log.With().Str("handler", "postHandler").Logger()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			log.Warn().Msg("Method not allowed")
			return
		}
		file, info, err := r.FormFile(config.FormField)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Error while uploading"))
			log.Warn().Err(err).Msg("Error while uploading")
			return
		}
		defer file.Close()
		if info.Size > config.MaxFileSizeMB*1024*1024 {
			log.Warn().Int64("maxFileSizeMB", config.MaxFileSizeMB).Int64("fileSize", info.Size).Msg("File size is too big")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			w.Write([]byte("Error file size is too big"))
			return
		}

		body, err := ioutil.ReadAll(file)
		if err != nil {
			log.Warn().Err(err).Msg("failed to read the uploaded file")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error while uploading"))
			return
		}
		filename := info.Filename
		if filename == "" {
			log.Warn().Err(err).Msg("filename is empty")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Filename is empty"))
			return
		}

		dstPath := path.Join(config.BasePath, filename)
		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			log.Warn().Err(err).Msg("failed to create the file")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error could not create the file"))
			return
		}
		defer dstFile.Close()

		if _, err := dstFile.Write(body); err != nil {
			log.Warn().Err(err).Msg("failed to write the file")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error while writing the file"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("File uploaded successfully"))
	}
}
