package node

import (

	// bibs padrão

	"context"
	"errors"
	"fmt"
	"io"
	// "log"
	"os"
	"time"
	"strings"

	// bibs externas

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/ipfs/kubo/repo"
	"github.com/libp2p/go-libp2p/core/peer"

	iface "github.com/ipfs/kubo/core/coreiface"
)

// Estruturas e Setup //

type NodeState int

// Enum de estado

const (
	FOLLOWER NodeState = iota // 0
	CANDIDATE
	LEADER
)

// Vetor de dados

type Vector struct {
	Ver int
	Content []string
}

// No

type Node struct {
	IpfsCore *core.IpfsNode 
	IpfsApi  iface.CoreAPI
	CidVector Vector
	VectorCache map[int]Vector
	State NodeState
}


// Novo repositótio IPFS
func newIpfsRepo(repoPath string) (error) {

	_, err := os.Stat(repoPath)

	// err == nil -> retornou info sobre o caminho -> caminho já existente	
	if(err == nil){
		err = errors.New("Dirétoria/Repositório já existente\n") 
		return err
	}

	// Se existe outro tipo de erro para além do caminho não existir retornamos o erro
	if(!os.IsNotExist(err)){
		return err
	}


	// Criar a diretória
	err = os.Mkdir(repoPath, 0750)

	if(err != nil){
		err = fmt.Errorf("Falha ao criar diretoria: %v\n",err)
		return err
	}
	// Criar a configuração do repositório.
	cfg, err := config.Init(io.Discard, 2048)
	cfg.Pubsub.Router="gossipsub"
	cfg.Pubsub.Enabled=1
	cfg.Ipns.UsePubsub=1


	if err != nil {
		err = fmt.Errorf("Falha ao criar configuração: %v\n",err)
		return err
	}


	pluginInjection(repoPath)


	// Inicializar repositório
	err = fsrepo.Init(repoPath, cfg)

	if err != nil {
		err = fmt.Errorf("Falha ao iniciar: %v\n",err)
		return err
	}

	return nil

}

// Novo nó ipfs
func newIpfsNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {

	var repo repo.Repo	

	err := pluginInjection(repoPath)

	if(err == nil){
		repo, err = fsrepo.Open(repoPath)
	}

	// Erro
	if err != nil {

		//Criado novo repositório
		err = newIpfsRepo(repoPath)
		if err != nil {
			return nil, fmt.Errorf("Falha ao criar novo repositório : %v\n",err)
		}

		//Tenta abrir o repositório que esta na pasta repoPath
		repo, err = fsrepo.Open(repoPath)

		if err != nil {
			return nil, fmt.Errorf("Falha ao abrir novo repositótio : %v\n", err)
		}
	}



	// Defenições especificas do nó
	nodeOptions := &core.BuildCfg{
		Online:    true,
		Routing:   libp2p.DHTOption,
		Repo:      repo,
		Permanent: true,
		ExtraOpts: map[string]bool{
			"pubsub": true,
		},
	}


	// Criação do nó
	ipfsCore, err := core.NewNode(ctx, nodeOptions)

	if err != nil {
		return nil, fmt.Errorf("Falha ao criar nó IPFS: %v\n", err)
	}

	return ipfsCore, nil

}

// Connectar nó a peers
func connectToPeers(ipfs iface.CoreAPI, peers []string, self string) error {

	// Array para gerir nós desconectados	
	unconnectedPeers := make(map[string]error)	


	if(len(peers) > 0){

		for _, peerIdString := range peers {

			if(peerIdString == self){
				continue
			}

			// Descodificar o CID
			peerId, err := peer.Decode(peerIdString)
			addr := peer.AddrInfo{ID:peerId}

			if(err != nil){
				err = fmt.Errorf("Peer String Invalida: %v\n",err)
				unconnectedPeers[peerIdString] = err
				continue
			}

			timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel() 


			// Conectar com 1 peer
			err = ipfs.Swarm().Connect(timeoutCtx, addr)


			if(err != nil){
				err = fmt.Errorf("Conexão não foi possivel: %v\n",err)
				unconnectedPeers[peerIdString] = err
			}

		}

	}
	
	// 1 ou + peers não conectados com sucesso -> erro
	if len(unconnectedPeers) > 0 {

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%d peers não conectados:\n\n", len(unconnectedPeers)))

		for peerID, err := range unconnectedPeers {
			sb.WriteString(fmt.Sprintf(" - %s → %v\n", peerID, err))
		}

		return fmt.Errorf("%s",sb.String())
	}

	return nil


}


// Injeção de plugins (sem isto dá erro: unknown datastore type: flatfs)
// https://discuss.ipfs.tech/t/fixed-unknown-datastore-type-flatfs/15805/5
func pluginInjection(repoPath string)(error){

	plugins, err := loader.NewPluginLoader(repoPath)
	if err != nil {
		err = fmt.Errorf("Erro ao carregar plugins: %s", err)
		return err
	}

	if err := plugins.Initialize(); err != nil {

		err = fmt.Errorf("Erro ao inicializar plugins: %s", err)
		return err
	}

	if err := plugins.Inject(); err != nil {
		err = fmt.Errorf("Erro ao injetar plugins: %s", err)
		return err
	}

	return nil

}




// Criar novo nó

// Para colocar o nó a leader basta ativar esta flag
var LeaderFlag bool


func Create(repoPath string,  peers []string) (*Node,error){

	ipfsCore, err := newIpfsNode(context.Background(),repoPath);

	if(err!=nil){
		return nil,err
	}

	ipfsCoreApi,err := coreapi.NewCoreAPI(ipfsCore)


	if(err!=nil){
		return nil,err
	}

	if(peers != nil && cap(peers) > 0){
		err = connectToPeers(ipfsCoreApi,peers,ipfsCore.Identity.String())
	}

	state := FOLLOWER

	if(LeaderFlag){
		state = LEADER
	}

	n := Node {

		IpfsCore : ipfsCore,
		IpfsApi : ipfsCoreApi,
		VectorCache: make(map[int]Vector),
		CidVector: 
			Vector {
				Ver: 0,
				Content: []string{},
			},
		State: state,

	}

	return &n,err

}

// Funções auxiliares //

// Upload de ficheiros
func AddFile(nd *Node,fileBytes []byte) (path.ImmutablePath,error){


	uploadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fileCid, err := nd.IpfsApi.Unixfs().Add(uploadCtx,files.NewBytesFile(fileBytes))

	if(err != nil){
		err = fmt.Errorf("Erro ao dar upload de ficheiro: %v\n",err)
	}


	return fileCid,err

}




