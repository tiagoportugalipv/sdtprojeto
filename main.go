package main

import (

	// bibs padrão
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"


	// bibs internas
	"projeto/api"
	"projeto/node"
	"projeto/services/embedding"

)

func main() {

	var repoPath string
	var peersFilePath string
	var modelsDirPath string
	var leaderFlag bool
	var peers []string
	var nd *node.Node
	var err error

	flag.StringVar(&repoPath,"r",".ipfs","repositório ipfs")
	flag.BoolVar(&leaderFlag,"l",false,"Iniciar o nó a lider [para testes]")
	flag.StringVar(&peersFilePath,"p","peers.txt","ficheiro de peers, se não for necessário passe \"null\"")
	flag.StringVar(&modelsDirPath,"m","models","diretoria de modelos")

	flag.Parse()

	node.LeaderFlag = leaderFlag

	repoPath,err = filepath.Abs(repoPath)

	if(err != nil){
		err = fmt.Errorf("Caminho para repositório ipfs invalido %v",err)
		panic(err)
	}


	modelsDirPath,err = filepath.Abs(modelsDirPath)

	if(err != nil){
		err = fmt.Errorf("Caminho para repositório ipfs invalido %v",err)
		panic(err)
	}


	peers = nil

	if( peersFilePath != "null" ){


		peersFilePath,err = filepath.Abs(peersFilePath)

		if(err != nil){
			err = fmt.Errorf("Caminho para repositório ipfs invalido %v",err)
			panic(err)
		}

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

	fmt.Println("==> Iniciar setup do modelo de embedding")

	embedding.ModelDirPath = modelsDirPath
	err,_ = embedding.SetUpModel()

	if(err != nil){
		err = fmt.Errorf("Erro ao dar setup do modelo de embedding: %v",err)
		panic(err)
	}

	fmt.Printf("\tModelo iniciado com sucesso\n\n")


	fmt.Printf("==> A criar nó\n\n")

	nd,err = node.Create(repoPath,peers)

	if(err != nil){
		fmt.Printf("\nErro(s) ao criar nó: %v\n",err)
	}

	if(nd == nil){
		fmt.Printf("Nó não foi criado com sucesso\n")
		os.Exit(1)
	}


	fmt.Printf("\tNó criado com sucesso\n\n")


	apiInterface := api.APIInterface{NodeId: nd.IpfsCore.Identity.String()}
	nd.SetAPI(&apiInterface)

	nd.Run()

}

