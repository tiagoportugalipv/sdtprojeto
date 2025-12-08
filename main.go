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

	// bibs externas
)

func main() {

	var repoPath string
	var peersFilePath string
	var modelsDirPath string
	var leaderFlag bool
	var peers []string
	var nd *node.Node
	var err error
	var apiPort = 9000

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

	// Leitura de peers para criação do nó

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


	// Teste de funcionamento do modelo de embeddings

	// embs,err := embedding.GetEmbeddings(peers)
	//
	// dimension := len(embs[0])
	// fmt.Printf("Número de linhas: %d\n", len(embs))
	// fmt.Printf("Dimensão dos embeddings: %d\n", dimension)


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

	// Teste de funcionamento do IPFS

	// 	file, err := os.Open("files/sample.md")
	// 	fileBytes, err := io.ReadAll(file)
	// 
	// 	fileCid, err := node.AddFile(nd,fileBytes)
	//
	// 	fmt.Printf("Cid: %v",fileCid)
	// 	file.Close()

	// Teste de funcionamento do PubSub

	// if(leaderFlag){
	//
	//
	// 	fmt.Printf("==>A correr Api na porta %d\n",apiPort)
	//
	// 	err := messaging.PublishTo(nd.IpfsApi.PubSub(),messaging.TXT,messaging.TextMessage{Text: "Olá eu sou o lider"})
	//
	//
	// 	if(err != nil){
	// 		fmt.Printf("Mensagem não enviada com sucesso: v%\n",err)
	// 	}
	//
	// 	err = api.Initialize(nd,apiPort)
	//
	// 	if(err != nil){
	// 		fmt.Printf("Api não foi iniciada com sucesso\n")
	// 	}
	//
	// } else {
	//
	//
	// 	fmt.Printf("==>A ouvir mensagens\n")
	//
	// 	err = messaging.ListenTo(nd.IpfsApi.PubSub(),messaging.TXT,func(sender peer.ID, msg any) {
	//
	// 		textMsg,ok := msg.(messaging.TextMessage)
	//
	// 		if(!ok){
	// 			fmt.Printf("Esperado TextMessage, obtido %T", msg)
	// 		}
	//
	// 		fmt.Println(textMsg.Text)
	// 	})
	//
	//
	// 	if(err != nil){
	// 		fmt.Printf("Mensagem não recebida com sucesso: v%\n",err)
	// 	}
	//
	// }

	if(leaderFlag){
		go api.Initialize(nd,apiPort)
	}

	nd.Run()

}

