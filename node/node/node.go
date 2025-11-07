package node

// bibs padrão
import (
	"errors"
	"io"
	"os"
	"fmt"
	"context"
	"time"

)

// bibs externas
import(

	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/ipfs/kubo/plugin/loader"
	iface "github.com/ipfs/kubo/core/coreiface"

)

type Vector struct {
	//ver int
	Content []string

}

type Node struct {
	IpfsCore *core.IpfsNode 
	IpfsApi  iface.CoreAPI
	CidVector Vector
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


	if err != nil {
		err = fmt.Errorf("Falha ao criar configuração: %v\n",err)
		return err
	}


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

	repo, err := fsrepo.Open(repoPath)

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

	// Injeção de plugins (sem isto dá erro)


	plugins, err := loader.NewPluginLoader(repoPath)

	if err != nil {
		//indica um erro e interrompe a execução do ficheiro
		panic(fmt.Errorf("error loading plugins: %s\n", err))
	}

	if err := plugins.Initialize(); err != nil {
		panic(fmt.Errorf("error initializing plugins: %s\n", err))
	}

	if err := plugins.Inject(); err != nil {

		panic(fmt.Errorf("error initializing plugins: %s\n", err))
	}



	// Defenições especificas do nó
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption,
		Repo:    repo,
	}


	// Criação do nó
	ipfsCore, err := core.NewNode(ctx, nodeOptions)

	if err != nil {
		return nil, fmt.Errorf("Falha ao criar nó IPFS: %v\n", err)
	}

	return ipfsCore, nil

}

// Connectar nó a peers
func connectToPeers(ipfs iface.CoreAPI, peers []string) error {

	// Array para gerir nós desconectados	
	unconnectedPeers := make(map[string]error)	


	if(len(peers) > 0){

		for _, peerIdString := range peers {

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
	if(len(unconnectedPeers)>0){
		err := fmt.Errorf("%v não conectados : %v\n",len(unconnectedPeers),unconnectedPeers)
		return err
	}

	return nil


}


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
		err = connectToPeers(ipfsCoreApi,peers)
	}

	n := Node{
		IpfsCore : ipfsCore,
		IpfsApi : ipfsCoreApi,
		CidVector: 
			Vector{
				Content: []string{},
			},

	}

	return &n,err

}
