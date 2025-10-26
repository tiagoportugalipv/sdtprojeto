package controllers

import (
	"context"
	"net/http"
    "io"
    "time"

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

    openedFile, err := file.Open()
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
        return
    }
    defer openedFile.Close()

    fileBytes, err := io.ReadAll(openedFile)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
        return
    }

    uploadCtx, cancel := context.WithTimeout(nodeCtx, 30*time.Second)
    defer cancel()

    peerCidFile, err := ipfs.Unixfs().Add(uploadCtx, files.NewBytesFile(fileBytes))
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add file to IPFS: " + err.Error()})
        return
    }

    ctx.JSON(http.StatusOK, gin.H{
        "message": "File added successfully, CID : "+peerCidFile.String(),
        "filename": file.Filename,
        "cid": peerCidFile.String(),
    })
}

