package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type Block struct {
	Index     int
	Timestamp string
	BPM       int
	Hash      string
	PrevHash  string
}

type message struct {
	BPM int
}

var Blockchain []Block

func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func generateBlock(oldBlock Block, BPM int) (Block, error) {
	var newBlock Block

	t := time.Now()
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Index = oldBlock.Index + 1
	newBlock.BPM = BPM
	newBlock.Timestamp = t.String()
	newBlock.Hash = calculateHash(newBlock)

	return newBlock, nil
}

func isBlockValidate(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
}

func run() error {
	mux := makeMuxRouter()
	httpAddr := os.Getenv("PORT")
	log.Println("Listening on ", httpAddr)
	s := &http.Server{
		Addr:           ":" + httpAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("Get")
	muxRouter.HandleFunc("/", handleWriteBlock).Methods("Post")
	return muxRouter
}

func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", " ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(bytes))
}

func handleWriteBlock(w http.ResponseWriter, r *http.Request) {
	var m message

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		responseWithJson(w, r, http.StatusBadRequest, r.Body)
		return
	}
	defer r.Body.Close()

	oldBlock := Blockchain[len(Blockchain)-1]
	newBlock, err := generateBlock(oldBlock, m.BPM)
	if err != nil {
		responseWithJson(w, r, http.StatusInternalServerError, r.Body)
		return
	}

	if isBlockValidate(newBlock, oldBlock) {
		newBlockChain := append(Blockchain, newBlock)
		replaceChain(newBlockChain)
		spew.Dump(Blockchain)
	}

	responseWithJson(w, r, http.StatusCreated, newBlock)
}

func responseWithJson(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
	response, err := json.MarshalIndent(payload, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP: 500"))
		return
	}
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		t := time.Now()
		gensisBlock := Block{
			Index:     0,
			Timestamp: t.String(),
			BPM:       0,
			Hash:      "",
			PrevHash:  "",
		}
		spew.Dump(gensisBlock)
		Blockchain = append(Blockchain, gensisBlock)
	}()
	log.Fatal(run())
}
