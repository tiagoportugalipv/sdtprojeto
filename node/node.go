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
	"github.com/libp2p/go-libp2p/core/peer"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// createRepo garante que o repositório IPFS existe no caminho indicado.
// Devolve true em caso de sucesso (existente ou criado) e false se falhar.
func createRepo(repoPath string) bool {

	_, err := os.Stat(repoPath)
	if err == nil {
		return true
	}

	if !os.IsNotExist(err) {
		log.Printf("Failed to check repo directory: %v", err)
		return false
	}

	err = os.Mkdir(repoPath, 0750)
	if err != nil {
		log.Printf("Failed to create repo directory: %v", err)
		return false
	}

	cfg, err := config.Init(io.Discard, 2048) // configuração inicial do repo
	if err != nil {
		log.Printf("Failed to create repo config: %v", err)
		return false
	}

	err = fsrepo.Init(repoPath, cfg) // inicializa o repositório no disco
	if err != nil {
		log.Printf("Failed to initialize repo: %v", err)
		return false
	}

	return true
}

// createNode abre (ou cria) o repositório e instancia um nó IPFS online.
func createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {

	repo, err := fsrepo.Open(repoPath)

	if err != nil {
		if !createRepo(repoPath) {
			return nil, fmt.Errorf("failed to create repo")
		}
		repo, err = fsrepo.Open(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open repo after creation: %v", err)
		}
	}

	nodeOptions := &core.BuildCfg{ // configuração do nó IPFS
		Online: true,               // opera em modo online
		Routing: libp2p.DHTOption,  // usa DHT para descoberta/roteamento
		Repo: repo,                 // repositório associado
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create IPFS node: %v", err)
	}

	// Ativa descoberta mDNS para facilitar a ligação automática entre peers na LAN
	if err := enableMdnsDiscovery(ctx, node); err != nil {
		log.Printf("mDNS discovery error: %v", err)
	}

	return node, nil
}

// notifeeMdns tenta ligar-se a peers descobertos via mDNS
type notifeeMdns struct{ node *core.IpfsNode }

func (n *notifeeMdns) HandlePeerFound(pi peer.AddrInfo) {
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

func enableMdnsDiscovery(ctx context.Context, node *core.IpfsNode) error {
	service := mdns.NewMdnsService(node.PeerHost, "sdt-mdns", &notifeeMdns{node: node})
	return service.Start()
}


// main: carrega plugins, cria o nó IPFS, inicializa Pub/Sub e API HTTP
func main() {
	repoPath := "C:\\Users\\bento\\.ipfs"

	plugins, err := loader.NewPluginLoader(repoPath) // gestor de plugins do IPFS
	if err != nil {
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

	// outputPath := "randomFile.txt"
	//
	// cidFicheiro,_ := cid.Decode("QmYaBU4DJyEZCntf2Pua2kdgnXDz7gntSzuUwxer4XgAX9")
	// ficheiro,_ := ipfsService.Unixfs().Get(ctx,path.FromCid(cidFicheiro));
	//
	// files.WriteTo(ficheiro,outputPath)

	fmt.Println("IPFS Node created successfully: "+node.Identity.String())

	pubSubService, err := messaging.NewPubSubService(ctx, node, "uploaded") // serviço Pub/Sub

	if err != nil {
		log.Fatalf("Error creating PubSub service: %v", err)
	}

	log.Printf("Subscribed to topic: %s as %s", "uploaded", node.Identity.String()) // confirmação

	// Publica uma mensagem de presença ao iniciar (ajuda a validar receção entre peers)
	if err := pubSubService.PublishMessage("Node online: " + node.Identity.String()); err != nil {
		log.Printf("Falha ao publicar mensagem de presença: %v", err)
	}
	api.Initialize(ctx, ipfsService, pubSubService) // arranca a API HTTP (porta 9000)
}
