package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/jsonq"
	"github.com/vjeantet/jodaTime"

	"main/db"
	"main/jwt"
	"main/mail"
	"main/ptt"
)

func sendDailyBeauty(subscribers []string, isTest bool) {
	log.Println("getting daily beauty...")
	// TODO: do parallelly
	beauties, err := ptt.FetchBeauties()
	if err != nil {
		panic(err)
	}

	randomBeauty, err := ptt.FetchRandomBeauty()
	if err != nil {
		panic(err)
	}

	loc, _ := time.LoadLocation("Asia/Taipei")
	date := jodaTime.Format("YYYY-MM-dd", time.Now().In(loc))
	subject := fmt.Sprintf("表特日報-%s", date)

	if isTest {
		subject += " " + strconv.Itoa(rand.Int())
	}

	log.Println("sending...")
	for _, to := range subscribers {
		token := jwt.NewToken(to)
		html := mail.GenerateHTML(beauties, randomBeauty, token)
		mail.Send(to, subject, html)
		log.Printf("Send to '%s' success", to)
		time.Sleep(200 * time.Millisecond)
	}

	log.Println("Finish")
}

func testHandler(c *gin.Context) {
	toMails := []string{"flaviogptdb@gmail.com"}
	sendDailyBeauty(toMails, true)
	log.Printf("Test successfully\n")
	c.String(200, "Test successfully")
}

func publishHandler(c *gin.Context) {
	toMails, err := db.GetEmails()
	if err != nil {
		panic(err)
	}

	sendDailyBeauty(toMails, false)
	log.Printf("Publish to %d users successfully\n", len(toMails))
	c.String(200, "Publish successfully")
}

func subscribeHandler(c *gin.Context) {
	data := map[string]interface{}{}
	dec := json.NewDecoder(c.Request.Body)
	dec.Decode(&data)
	jq := jsonq.NewQuery(data)
	email, err := jq.String("email")
	if err != nil {
		panic(err)
	}
	db.InsertAEmail(email)
}

// api/unsubscribe?token={jwt_token}
func unsubscribeHandler(c *gin.Context) {
	tokenStr := c.Query("token")
	email, err := jwt.ParseToken(tokenStr)

	if err != nil {
		c.AbortWithError(400, err)
		// TODO: render error to frontend
		return
	}

	db.RemoveAEmail(email)
	log.Printf("%s unsubscribe", email)
	c.String(200, "you(%s) have been unsubscribed from our mailing list", email)
}

func emailsHandler(c *gin.Context) {
	emails, err := db.GetEmails()
	if err != nil {
		panic(err)
	}
	c.JSON(200, emails)
}

func homePageHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

// Redirect /ptt/redirect/M.1543991133.A.1A1
// to https://www.ptt.cc/bbs/Beauty/M.1543991133.A.1A1.html
func pttRedirectHandler(c *gin.Context) {
	baseURL := "https://www.ptt.cc/bbs/Beauty/"
	articleID := c.Param("articleID")
	location := fmt.Sprintf("%s%s.html", baseURL, articleID)
	c.Redirect(302, location)
}

func main() {
	r := gin.Default()
	r.GET("/test", testHandler)
	r.GET("/publish", publishHandler)

	r.POST("/api/subscribe", subscribeHandler)
	r.GET("/api/unsubscribe", unsubscribeHandler)
	r.GET("/api/emails", emailsHandler)

	r.GET("/ptt/redirect/:articleID", pttRedirectHandler)

	r.LoadHTMLFiles("index.html")
	r.GET("/", homePageHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listen on port %s", port)
	err := r.Run(":" + port)
	panic(err)
}

// TODO: analysis 轉網址
// TODO: 禮拜幾標題變化
// TODO: 下載所有圖片
// TODO: 防止手動觸發 cron
