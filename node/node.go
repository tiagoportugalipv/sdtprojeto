// Programa principal do nó: inicializa o repositório IPFS, cria o nó,
// configura Pub/Sub e arranca a API HTTP.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"sdt/node/api"
	"sdt/node/services/messaging"

	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
	libpeer "github.com/libp2p/go-libp2p/core/peer"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// createRepo garante que o repositório IPFS existe no caminho indicado.
// Devolve true em caso de sucesso (existente ou criado) e false se falhar.
func createRepo(repoPath string) bool {

	//Ignora o primeiro valor devolvido e de seguida verifica se o ficheiro existe
	_, err := os.Stat(repoPath)

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

// Esta função serve para criar um nó IPFS a partir de um repositório existente
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
		Online:  true,
		Routing: libp2p.DHTOption,
		Repo:    repo,
	}

	//Cria o nó IPFS com as opções definidas anteriomente
	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create IPFS node: %v", err)
	}

	// Ativa descoberta mDNS para facilitar a ligação automática entre peers na LAN
	// mDNS publica o meu hostname na rede
	if err := enableMdnsDiscovery(node); err != nil {
		log.Printf("mDNS discovery error: %v", err)
	}

	return node, nil
}

// notifeeMdns tenta ligar-se a peers descobertos via mDNS
type notifeeMdns struct{ node *core.IpfsNode }

func (n *notifeeMdns) HandlePeerFound(pi libpeer.AddrInfo) {
	if pi.ID == n.node.Identity {
		return
	}
	go func() {
		if err := n.node.PeerHost.Connect(context.Background(), pi); err != nil {
			log.Printf("Failed to connect to discovered peer %s: %v", pi.ID.String(), err)
		} else {
			log.Printf("Connected to discovered peer: %s", pi.ID.String())
		}
	}()
}

func enableMdnsDiscovery(node *core.IpfsNode) error {
	service := mdns.NewMdnsService(node.PeerHost, "sdt-mdns", &notifeeMdns{node: node})
	return service.Start()
}

// main: carrega plugins, cria o nó IPFS, inicializa Pub/Sub e API HTTP
func main() {
	repoPath := "C:\\Users\\admin\\.ipfs"

	cidVector := []string{} //new - slice para guardar CIDs

	plugins, err := loader.NewPluginLoader(repoPath) // gestor de plugins do IPFS
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

	ipfsService, err := coreapi.NewCoreAPI(node) // CoreAPI de alto nível para IPFS
	if err != nil {
		log.Fatalf("Error creating IPFS CoreAPI: %v", err)
	}

	// Conexão a peers é automática via mDNS (LAN) e através da rede IPFS/libp2p padrão

	// outputPath := "randomFile.txt"
	//
	// cidFicheiro,_ := cid.Decode("QmYaBU4DJyEZCntf2Pua2kdgnXDz7gntSzuUwxer4XgAX9")
	// ficheiro,_ := ipfsService.Unixfs().Get(ctx,path.FromCid(cidFicheiro));
	//
	// files.WriteTo(ficheiro,outputPath)

	fmt.Println("IPFS Node created successfully: " + node.Identity.String())

	pubSubService, err := messaging.NewPubSubService(ctx, node, "batatas") // serviço Pub/Sub

	if err != nil {
		log.Fatalf("Error creating PubSub service: %v", err)
	}

	log.Printf("Subscribed to topic: %s as %s", "batatas", node.Identity.String()) // confirmação

	// Publica uma mensagem de presença ao iniciar
	/*if err := pubSubService.PublishMessage("Node online: " + node.Identity.String()); err != nil {
		log.Printf("Falha ao publicar mensagem de presença: %v", err)
	}*/

	// Loop de consola: cada linha escrita será publicada no tópico
	// go func() {
	// 	scanner := bufio.NewScanner(os.Stdin)
	// 	fmt.Println("Type a message and press Enter to publish to 'batatas':")
	// 	for scanner.Scan() {
	// 		text := scanner.Text()
	// 		if text == "" {
	// 			continue
	// 		}
	// 		if err := pubSubService.PublishMessage(text); err != nil {
	// 			log.Printf("Failed to publish from console: %v", err)
	// 		} else {
	// 			log.Printf("Published from console: %s", text)
	// 		}
	// 	}
	// 	if err := scanner.Err(); err != nil {
	// 		log.Printf("Console read error: %v", err)
	// 	}
	// }()
	api.Initialize(ctx, ipfsService, pubSubService, cidVector) // arranca a API HTTP (porta 9000)
}

// connectToPeers tenta ligar aos multiaddrs fornecidos usando a CoreAPI (Swarm.Connect)
// ligação manual via variável de ambiente removida para manter descoberta automática
