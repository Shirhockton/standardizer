package controllers

import (
	"net/http"
	"standardizer/global"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

func LikeArticle(ctx *gin.Context) {
	articleID := ctx.Param("id")

	likeKey := "article:" + articleID + ":likes"
	if err := global.RedisDB.Incr(likeKey).Err(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Successfully liked article"})
}

func GetArticleLikes(ctx *gin.Context) {
	articleID := ctx.Param("id")

	likeKey := "article:" + articleID + ":likes"
	likes, err := global.RedisDB.Get(likeKey).Result()
	if err == redis.Nil {
		likes = "0"
	} else if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else {
		/* no deal with */
	}
	ctx.JSON(http.StatusOK, gin.H{"likes": likes})

}
