package io

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs/boxo/files"
	iface "github.com/ipfs/kubo/core/coreiface"
)

func UploadFile(ctx *gin.Context, nodeCtx context.Context, ipfs iface.CoreAPI){

    file, err := ctx.FormFile("file")
    
    if err != nil || file == nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
        return
    }


    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
        return
    }

    fileBytes := make([]byte, file.Size)
    peerCidFile, err := ipfs.Unixfs().Add(nodeCtx,files.NewBytesFile(fileBytes))


    ctx.JSON(http.StatusOK, gin.H{
        "message": "File added successfully, CID : "+peerCidFile.String(),
        "filename": file.Filename,
    })
}

