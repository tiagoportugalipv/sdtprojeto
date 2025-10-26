package file 


import (
 "net/http"
 "github.com/gin-gonic/gin"
)

func UploadFile(ctx *gin.Context){

    file, err := ctx.FormFile("file")
    
    if err != nil || file == nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
        return
    }

    if err := ctx.SaveUploadedFile(file, "./uploads/"+file.Filename); err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "unable to save file"})
        return
    }

    ctx.JSON(http.StatusOK, gin.H{
        "message": "File received successfully",
        "filename": file.Filename,
    })
}

