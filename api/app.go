package main

import (
  "github.com/gin-gonic/gin"
  "github.com/joho/godotenv"
  "os"
  fileRoutes "sdt/api/routes/file"
)

func main(){

  err := godotenv.Load()
  if err != nil {
    os.Stderr.WriteString("Error loading .env file");
  }

  app := gin.New();
  app.Use(gin.Logger());
  app.Use(gin.Recovery());


  fileRoutes.SetUpRoutes(app.Group("/file"));
  app.Run(":9000");

}
