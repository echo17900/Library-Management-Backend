package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-ini/ini"
	_ "github.com/go-sql-driver/mysql"
	. "lms/services"
	"lms/util"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

var agent DBAgent

func loginHandler(context *gin.Context) {
	userid := context.PostForm("userid")
	password := context.PostForm("password")
	loginResult, userID := agent.AuthenticateUser(userid, password)
	if loginResult.Status == UserLoginOK {
		token := util.GenToken(userID, util.UserKey)
		context.JSON(http.StatusOK, gin.H{"status": loginResult.Status, "msg": loginResult.Msg, "token": token})
	} else {
		context.JSON(http.StatusOK, gin.H{"status": loginResult.Status, "msg": loginResult.Msg})
	}
}

func adminLoginHandler(context *gin.Context) {
	username := context.PostForm("username")
	password := context.PostForm("password")
	loginResult, userID := agent.AuthenticateAdmin(username, password)
	if loginResult.Status == AdminLoginOK {
		token := util.GenToken(userID, util.AdminKey)
		context.JSON(http.StatusOK, gin.H{"status": loginResult.Status, "msg": loginResult.Msg, "token": token})
	} else {
		context.JSON(http.StatusOK, gin.H{"status": loginResult.Status, "msg": loginResult.Msg, "token": ""})
	}
}

func getCountHandler(context *gin.Context) {
	bookCount := agent.GetBookNum()
	context.JSON(http.StatusOK, gin.H{"count": bookCount})
}

func getBooksHandler(context *gin.Context) {
	pageString := context.PostForm("page")
	page, _ := strconv.Atoi(pageString)
	books := agent.GetBooksByPage(page)

	bf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(bf)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(books)

	_, _ = context.Writer.Write(bf.Bytes())
}

func getBorrowTimeHandler(context *gin.Context) {
	bookIDString := context.PostForm("bookID")
	bookID, _ := strconv.Atoi(bookIDString)
	UserIDString := context.PostForm("userID")
	userID, _ := strconv.Atoi(UserIDString)
	subTime := agent.GetBorrowTime(bookID, userID)

	bf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(bf)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(subTime)

	_, _ = context.Writer.Write(bf.Bytes())
}

func getUserBooksHandler(context *gin.Context) {
	iUserID, _ := context.Get("userID")
	userID := iUserID.(int)
	pageString := context.PostForm("page")
	page, _ := strconv.Atoi(pageString)
	books := agent.GetUserBooksByPage(userID, page)

	bf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(bf)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(books)

	_, _ = context.Writer.Write(bf.Bytes())
}

func getReserveBooksHandler(context *gin.Context) { //数据类型待定（不知道是int还是string）,则前面的可能要改
	iUserID, _ := context.Get("userID")
	userID := iUserID.(int)
	bookIDString := context.PostForm("bookID")
	bookID, _ := strconv.Atoi(bookIDString)
	result := agent.ReserveBook(userID, bookID)
	context.JSON(http.StatusOK, gin.H{"status": result.Status, "msg": result.Msg})
}

func getCancelReserveBooksHandler(context *gin.Context) {
	iUserID, _ := context.Get("userID")
	userID := iUserID.(int)
	bookIDString := context.PostForm("bookID")
	bookID, _ := strconv.Atoi(bookIDString)
	result := agent.CancelReserveBook(userID, bookID)
	context.JSON(http.StatusOK, gin.H{"status": result.Status, "msg": result.Msg})
}

func borrowBookHandler(context *gin.Context) {
	iUserID, _ := context.Get("userID")
	userID := iUserID.(int)
	bookIDString := context.PostForm("bookID")
	bookID, _ := strconv.Atoi(bookIDString)
	result := agent.BorrowBook(userID, bookID)
	context.JSON(http.StatusOK, gin.H{"status": result.Status, "msg": result.Msg})
}

func returnBookHandler(context *gin.Context) {
	iUserID, _ := context.Get("userID")
	userID := iUserID.(int)
	bookIDString := context.PostForm("bookID")
	bookID, _ := strconv.Atoi(bookIDString)
	result := agent.ReturnBook(userID, bookID)
	context.JSON(http.StatusOK, gin.H{"status": result.Status, "msg": result.Msg})
}

func updateBookStatusHandler(context *gin.Context) {
	bookStatusString := context.PostForm("bookStatus")
	bookStatusMap := make(map[string]string)
	err := json.Unmarshal([]byte(bookStatusString), &bookStatusMap)
	if err != nil {
		log.Println(err.Error())
	}
	book := new(Book)
	book.Id, _ = strconv.Atoi(bookStatusMap["id"])
	book.Name = bookStatusMap["name"]
	book.Author = bookStatusMap["author"]
	book.Isbn = bookStatusMap["isbn"]
	book.Address = bookStatusMap["address"]
	book.Language = bookStatusMap["language"]
	book.Count, _ = strconv.Atoi(bookStatusMap["count"])
	result := agent.UpdateBookStatus(book)
	context.JSON(http.StatusOK, gin.H{"status": result.Status, "msg": result.Msg})
}

//addbook?isbn=&count=&location=
func addBookHandler(context *gin.Context) {
	var err error
	bookStatusString := context.PostForm("bookStatus")
	bookStatusMap := make(map[string]string)
	err = json.Unmarshal([]byte(bookStatusString), &bookStatusMap)
	if err != nil {
		log.Println(err.Error())
	}
	isbn := bookStatusMap["isbn"]
	count := bookStatusMap["count"]
	location := bookStatusMap["location"]
	var book Book

	book, err = GetMetaDataByISBN(isbn)
	if err != nil {
		log.Println("metadata retriever failure: " + err.Error())
		book.Name = "Unknown"
		book.Author = "Unknown"
		book.Language = "Unknown"
		book.Isbn = isbn
	}
	book.Count, _ = strconv.Atoi(count)
	book.Location = location
	result := agent.AddBook(&book)
	if result.Status == UpdateOK {
		log.Printf("Add Book %v (ISBN:%v) Successfully \n", book.Name, book.Isbn)
	} else {
		log.Printf("FAIL TO Add Book %v (ISBN:%v)  \n", book.Name, book.Isbn)
	}
	context.JSON(http.StatusOK, gin.H{"status": result.Status, "msg": result.Msg})
}

func deleteBookHandler(context *gin.Context) {
	bookID, err := strconv.Atoi(context.PostForm("bookID"))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	result := agent.DeleteBook(bookID)
	context.JSON(http.StatusOK, gin.H{"status": result.Status, "msg": result.Msg})
}

// ------Book BarCode Handle Section Start-------

// GetBookBarimg Handler:
// Method:GET param={id,isbn}
func getBookBarcodeImageHandler(context *gin.Context) {
	idString := context.Query("id")
	id, _ := strconv.Atoi(idString)
	isbn := context.Query("isbn")
	result, path := agent.GetBookBarcodePath(id, isbn)
	if result.Status == BookBarcodeFailed {
		log.Println(result.Msg)
		context.Data(http.StatusInternalServerError, "image/png", nil)
		return
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Println(err.Error())
			context.Data(http.StatusInternalServerError, "image/png", nil)
		}
		context.Data(http.StatusOK, "image/png", data)
	}
}

// ------Book BarCode Handle Section End-------
func loadConfig(configPath string) {
	Cfg, err := ini.Load(configPath)
	if err != nil {
		log.Fatal("Fail to Load config: ", err)
	}

	server, err := Cfg.GetSection("server")
	if err != nil {
		log.Fatal("Fail to load section 'server': ", err)
	}
	httpPort := server.Key("port").MustInt(80)
	path := server.Key("path").MustString("")
	staticPath := server.Key("staticPath").MustString("")
	Jikeapikey = server.Key("JiKeAPIKey").MustString("")

	mysql, err := Cfg.GetSection("mysql")
	if err != nil {
		log.Fatal("Fail to load section 'mysql': ", err)
	}
	username := mysql.Key("username").MustString("")
	password := mysql.Key("password").MustString("")
	address := mysql.Key("address").MustString("")
	tableName := mysql.Key("table").MustString("")

	db, err := sql.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=true", username, password, address, tableName))
	if err != nil {
		panic("connect to DB failed: " + err.Error())
	}
	agent.DB = db

	MediaPath = filepath.Join(path, "media")

	err = os.MkdirAll(MediaPath, os.ModePerm)
	if err != nil {
		log.Fatal("file system failed to create path: " + err.Error())
	}
	startService(httpPort, path, staticPath)

}

// 用户注册
func registerWithPasswordHandler(context *gin.Context) {
	userid := context.PostForm("userid")
	password := context.PostForm("password")
	email := context.PostForm("email")
	registerResult := agent.RegisterUserWithPassword(userid, password, email)
	if registerResult.Status != RegisterError {
		context.JSON(http.StatusOK, gin.H{"status": registerResult.Status, "msg": registerResult.Msg})
	} else {
		context.JSON(http.StatusInternalServerError, gin.H{"status": registerResult.Status, "msg": registerResult.Msg})
	}
}

// 获取用户二维码
func getUserBarcodeImageHandler(context *gin.Context) {
	idString := context.Query("id")
	id, _ := strconv.Atoi(idString)
	path, result := agent.GetUserBarcodePath(id)
	if result.Status == UserBarcodeFailed {
		log.Println(result.Msg)
		context.Data(http.StatusInternalServerError, "image/png", nil)
		return
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Println(err.Error())
			context.Data(http.StatusInternalServerError, "image/png", nil)
		}
		context.Data(http.StatusOK, "image/png", data)
	}
}

// 续借图书
func renewBookHandler(context *gin.Context) {
	iUserID := context.PostForm("userID")
	bookIDString := context.PostForm("bookID")
	borrowIDString := context.PostForm("borrowID")
	userID, _ := strconv.Atoi(iUserID)
	bookID, _ := strconv.Atoi(bookIDString)
	borrowID, _ := strconv.Atoi(borrowIDString)
	result := agent.RenewBook(borrowID, userID, bookID)
	if result.Status != RenewFailed {
		context.JSON(http.StatusOK, gin.H{"status": result.Status, "msg": result.Msg})
	} else {
		context.JSON(http.StatusInternalServerError, gin.H{"status": result.Status, "msg": result.Msg})
	}
}

//修改密码
func updatePasswordHandler(context *gin.Context) {
	oldpsw := context.PostForm("oldPassword")
	newpsw := context.PostForm("newPassword")
	userid := context.PostForm("userID")
	result := agent.UpdatePassword(oldpsw, newpsw, userid)
	if result.Status != UpdatePasswordFailed {
		context.JSON(http.StatusOK, gin.H{"status": result.Status, "msg": result.Msg})
	} else {
		context.JSON(http.StatusInternalServerError, gin.H{"status": result.Status, "msg": result.Msg})
	}
}

func startService(port int, path string, staticPath string) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	//router.LoadHTMLFiles(fmt.Sprintf("%v/index.html", path))
	//router.Use(static.Serve("/static", static.LocalFile(staticPath, true)))

	router.GET("/", func(context *gin.Context) {
		context.HTML(http.StatusOK, "index.html", nil)
	})
	//router.GET("/test", func(context *gin.Context) {
	//	context.String(http.StatusOK, "test")
	//})

	g1 := router.Group("/")
	g1.Use(middleware.UserAuth())
	{
		g1.POST("/getUserBooks", getUserBooksHandler)
		g1.POST("/getBorrowTime", getBorrowTimeHandler)
		g1.POST("/borrowBook", borrowBookHandler)
		g1.POST("/returnBook", returnBookHandler)
	}

	g2 := router.Group("/")
	g2.Use(middleware.AdminAuth())
	{
		g2.POST("/updateBookStatus", updateBookStatusHandler)
		g2.POST("/deleteBook", deleteBookHandler)
		g2.POST("/addBook", addBookHandler)
	}

	router.POST("/reserveBooks", getReserveBooksHandler)
	router.POST("/cancelReserveBooks", getCancelReserveBooksHandler)
	router.POST("/login", loginHandler)
	router.POST("/admin", adminLoginHandler)
	router.POST("/register", registerHandler)
	router.GET("/getCount", getCountHandler)
	router.GET("/getBooks", getBooksHandler)
	router.POST("/getBooks", getBooksHandler)
	router.GET("/getBookBarcode", getBookBarcodeImageHandler)

	g3 := router.Group("/pay")
	{
		g3.GET("/mobile", AliPayHandlerMobile)
		g3.GET("/pc", AliPayHandlerPC)
		g3.GET("/signcheck", SignCheck)
	}
	//router.StaticFile("/favicon.ico", fmt.Sprintf("%v/favicon.ico", staticPath))

	err := router.Run(":" + strconv.Itoa(port))
	if err != nil {
		fmt.Println(err)
		return
	} else {
		log.Println("running")
		return
	}
}

func main() {
	var configPath = flag.String("config", "./app.ini", "配置文件路径")
	flag.Parse()
	loadConfig(*configPath)
}
