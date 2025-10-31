package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
	"sdt/node/api"
	"sdt/node/services/messaging"
)


func createRepo(repoPath string) bool {

	//Ignora o primeiro valor devolvido e de seguida verifica se o ficheiro existe
	_, err := os.Stat(repoPath)

	//Se existe ficheiro
	if err == nil {
		return true
	}

	//true quando o erro é outro tipo de erro
	if !os.IsNotExist(err) {
		log.Printf("Failed to check repo directory: %v", err)
		return false
	}

	//Tenta criar o diretório, Se der erro, mostra uma mensagem no log e devolve falso
	err = os.Mkdir(repoPath, 0750)
	//Se der erro
	if err != nil {
		log.Printf("Failed to create repo directory: %v", err)
		return false
	}

	//Tenta criar a configuração do repositório.
	cfg, err := config.Init(io.Discard, 2048)
	//Se der erro
	if err != nil {
		log.Printf("Failed to create repo config: %v", err)
		return false
	}

	//inicializa o repositório no caminho repoPath usando a configuração cfg.
	//Cria a estrutura do repositório no disco	
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		log.Printf("Failed to initialize repo: %v", err)
		return false
	}

	return true
}

//Esta função serve para criar um nó IPFS a partir de um repositório existente
func createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {

	//Tenta abrir o repositório IPFS que está na pasta repoPath
	repo, err := fsrepo.Open(repoPath)

	//Se houver erro
	if err != nil {

		//Se não for criado o repositório
		if !createRepo(repoPath) {
			return nil, fmt.Errorf("failed to create repo")
		}
		//Tenta abrir o repositório que esta na pasta repoPath
		repo, err = fsrepo.Open(repoPath)

		if err != nil {
			return nil, fmt.Errorf("failed to open repo after creation: %v", err)
		}
	}

	//Prepara todas as definições que o nó IPFS precisa antes de ser criado
	nodeOptions := &core.BuildCfg{
		Online: true,
		Routing: libp2p.DHTOption,
		Repo: repo,
	}

	//Cria o nó IPFS com as opções definidas anteriomente
	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create IPFS node: %v", err)
	}

	return node, nil
}

//Esta função inicializa um nó IPFS com plugins e serviços associados, 
// preparando-o para enviar e receber mensagens via PubSub
func main() {
	repoPath := "C:\\Users\\admin\\.ipfs"

	plugins, err := loader.NewPluginLoader(repoPath)
	if err != nil {
		//indica um erro e interrompe a execução do ficheiro
		panic(fmt.Errorf("error loading plugins: %s", err))
	}

	if err := plugins.Initialize(); err != nil {
		panic(fmt.Errorf("error initializing plugins: %s", err))
	}

	if err := plugins.Inject(); err != nil {
		
		panic(fmt.Errorf("error initializing plugins: %s", err))
	}

	ctx := context.Background()
    
    node, err := createNode(ctx, repoPath)
    if err != nil {
        log.Fatalf("Error creating IPFS node: %v", err)
    } 

	ipfsService, err := coreapi.NewCoreAPI(node)
	if err != nil {
		log.Fatalf("Error creating IPFS CoreAPI: %v", err)
	}

	// outputPath := "randomFile.txt"
	//
	// cidFicheiro,_ := cid.Decode("QmYaBU4DJyEZCntf2Pua2kdgnXDz7gntSzuUwxer4XgAX9")
	// ficheiro,_ := ipfsService.Unixfs().Get(ctx,path.FromCid(cidFicheiro));
	//
	// files.WriteTo(ficheiro,outputPath)

	fmt.Println("IPFS Node created successfully: "+node.Identity.String())

	pubSubService, err := messaging.NewPubSubService(ctx, node, "uploaded")

	if err != nil {
		log.Fatalf("Error creating PubSub service: %v", err)
	}

	log.Printf("Subscribed to topic: %s as %s", "Uploaded-Files-Topic", node.Identity.String())

	if os.Getenv("LEADER") == "1" {
		if err := pubSubService.PublishMessage("Líder online: " + node.Identity.String()); err != nil {
			log.Printf("Falha ao publicar mensagem do líder: %v", err)
		}
	}
	api.Initialize(ctx, ipfsService, pubSubService)
}
