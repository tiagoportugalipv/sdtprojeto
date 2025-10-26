package file

import (
 "github.com/gin-gonic/gin"
 fileController "sdt/api/controllers/file"
)

func SetUpRoutes(rg *gin.RouterGroup){

  rg.POST("/upload", fileController.UploadFile )


}
