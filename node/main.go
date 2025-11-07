package main

// bibs padrão
import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"projeto/node"
	"strings"
	"io"
	"time"
)

// bibs internas
import (
	"github.com/ipfs/boxo/files"
)

func main() {

	var repoPath string
	var peersFilePath string
	var peers []string
	var nd *node.Node
	var err error

	flag.StringVar(&repoPath,"r",".ipfs","repositório ipfs")
	flag.StringVar(&peersFilePath,"p","peers.txt","repositório ipfs")

	fmt.Print(repoPath)

	peers = nil

	if( peersFilePath != "null" ){

		peers = []string{}

		peersFile, err := os.Open(peersFilePath)
		if err != nil {
			panic(fmt.Errorf("Não foi possivel abrir o ficheiro dos peers %v\n",err))
		}

		defer peersFile.Close()

		scanner := bufio.NewScanner(peersFile)

		for scanner.Scan() {
			peerIdString := strings.Trim(scanner.Text()," \n\t") 
			peers = append(peers,peerIdString) 
		}

	}

	nd,err = node.Create(repoPath,peers)

	if(err != nil){
		fmt.Printf("Erro(s) ao criar nó %v\n",err)
	}

	if(nd == nil){
		fmt.Printf("Nó não foi criado com sucesso\n")
		os.Exit(1)
	}

	file, err := os.Open("files/sample.md")
	fileBytes, err := io.ReadAll(file)


	uploadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//Envia o arquivo para o nó IPFS
	peerCidFile, err := nd.IpfsApi.Unixfs().Add(uploadCtx, files.NewBytesFile(fileBytes))

	fmt.Printf("Cid: %v",peerCidFile)
	file.Close()




}

