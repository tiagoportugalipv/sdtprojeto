package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
	"sdt/node/api"

)


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

	cfg, err := config.Init(io.Discard, 2048)
	if err != nil {
		log.Printf("Failed to create repo config: %v", err)
		return false
	}

	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		log.Printf("Failed to initialize repo: %v", err)
		return false
	}

	return true
}

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

	nodeOptions := &core.BuildCfg{
		Online: true,
		Routing: libp2p.DHTOption,
		Repo: repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create IPFS node: %v", err)
	}

	return node, nil
}


func main() {

	repoPath := "/home/tiago/.ipfs";

	plugins, err := loader.NewPluginLoader(repoPath)
	if err != nil {
		panic(fmt.Errorf("error loading plugins: %s", err))
	}

	if err := plugins.Initialize(); err != nil {
		panic(fmt.Errorf("error initializing plugins: %s", err))
	}

	if err := plugins.Inject(); err != nil {
		panic(fmt.Errorf("error initializing plugins: %s", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, err := createNode(ctx, repoPath)

	if err != nil {
		log.Fatalf("Error creating IPFS node: %v", err)
	} 


	ipfsService,err := coreapi.NewCoreAPI(node)


	// outputPath := "randomFile.txt"
	//
	// cidFicheiro,_ := cid.Decode("QmYaBU4DJyEZCntf2Pua2kdgnXDz7gntSzuUwxer4XgAX9")
	// ficheiro,_ := ipfsService.Unixfs().Get(ctx,path.FromCid(cidFicheiro));
	//
	// files.WriteTo(ficheiro,outputPath)

	fmt.Println("IPFS Node created successfully: "+node.Identity.String())

	api.Initialize(ctx, ipfsService)

	
}

