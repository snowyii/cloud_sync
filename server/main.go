package main

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"time"

	// "compress/gzip"

	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
)

// _ "github.com/go-sql-driver/mysql"

type Person struct {
	PID   string  `json:"pid"`
	HID   int     `json:"hid"`
	Name  *string `json:"name"`
	Birth *string `json:"birth"`
}

type Post struct {
	UUID        string  `json:"uuid"`
	AUTHOR      string  `json:"author"`
	MESSAGES    string  `json:"message"`
	LIKES       int     `json:"likes"`
	IMAGEUPDATE bool    `json:"imageUpdate"`
	IMAGE       *string `json:"image"`
}

type PostRes struct {
	UUID     string  `json:"u"`
	AUTHOR   *string `json:"a,omitempty"`
	MESSAGES *string `json:"m,omitempty"`
	LIKES    *int    `json:"l,omitempty"`
	IMG_UP   *uint8  `json:"i,omitempty"`
	DEL      int     `json:"d,omitempty"`
}

type Image struct {
	IMG string `json:"img"`
}

func main() {

	app := fiber.New(fiber.Config{
		BodyLimit:             1000 * 1000 * 1024 * 1024, // 1000MB
		DisableStartupMessage: false,
	})

	db, err := sql.Open("mysql", "root:Password123!@tcp(localhost:3307)/cloud")

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	err = db.Ping()

	if err != nil {
		log.Fatal(err)
	}
	//compressing

	app.Get("/api/messages/:last_sync", func(c *fiber.Ctx) error {
		conn, err := db.Conn(context.Background())
		if err != nil {
			log.Fatal(err)
		}

		defer conn.Close()

		last_sync := c.Params("last_sync")
		rows, err := conn.QueryContext(context.Background(),

			`SELECT uuid , 
		IF(DEL=1, NULL,author) as author, 
		IF(DEL=1,NULL,message) as message , 
		IF(DEL=1,NULL,likes) as likes , 
		IF(DEL=0 and img_last_update > ? , IF(image=1,1 , 2 ), NULL) as img_up , 
		del   
		FROM POST where last_update > ? ;`, last_sync, last_sync)

		if err != nil {
			log.Fatal((err))
		}
		defer rows.Close()

		posts := []PostRes{}

		for rows.Next() {
			var post PostRes
			err := rows.Scan(
				&post.UUID,
				&post.AUTHOR,
				&post.MESSAGES,
				&post.LIKES,
				&post.IMG_UP,
				&post.DEL,
			)

			if err != nil {
				log.Fatal(err)
				return c.Status(500).JSON("error")
			}

			posts = append(posts, post)

		}

		return c.JSON(posts)
	})

	app.Get("/api/last-sync", func(c *fiber.Ctx) error {
		var last_time string

		err := db.QueryRow("SELECT max(last_update) from POST").Scan(&last_time)
		if err != nil {
			return c.Status(500).SendString("")
		}

		return c.SendString(last_time)
	})
	app.Post("/api/messages", func(c *fiber.Ctx) error {

		// requestBody := c.Body()
		var post Post
		if err := c.BodyParser(&post); err != nil {
			fmt.Print(err)
			return c.Status(fiber.StatusBadRequest).JSON("Bad Request")

		}

		// fmt.Printf("===>%+v", post)
		imageUpdate := 0
		if post.IMAGEUPDATE {
			imageUpdate = 1
		}
		unixTimestamp := time.Now().Unix()

		_, err = db.Exec("INSERT INTO POST (uuid, author, message, likes, last_update, image, img_last_update) VALUES (?, ?, ?, ?, ?, ? , ?);", post.UUID, post.AUTHOR, post.MESSAGES, post.LIKES, unixTimestamp, imageUpdate, unixTimestamp)
		if err != nil {
			fmt.Println(err)
			return c.Status(409).SendString("")
		}

		// fmt.Println("post.IMAGE", *post.IMAGE)
		if imageUpdate == 1 {
			imageBytes, err := base64.StdEncoding.DecodeString(*post.IMAGE)
			if err != nil {
				// handle error
				fmt.Println(err)
			}
			file, err := os.OpenFile(fmt.Sprintf("dump/%s.jpg", post.UUID), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			if err != nil {
				// handle error
				fmt.Println(err)

			}
			defer file.Close()
			_, err = file.Write(imageBytes)
			if err != nil {
				// handle error
				log.Fatal(err)
			}
		}

		return c.Status(201).JSON("")
	})

	app.Get("/api/fetch/:uuid", func(c *fiber.Ctx) error {
		uuid := c.Params("uuid")

		binaryData, err := ioutil.ReadFile(fmt.Sprintf("dump/%s.jpg", uuid))
		if err != nil {
			panic(err)
		}

		return c.SendString(string(binaryData))
	})

	//only for test

	app.Put("/api/messages/:uuid", func(c *fiber.Ctx) error {

		uuid := c.Params("uuid")
		var post Post
		if err := c.BodyParser(&post); err != nil {
			fmt.Print(err)
			return c.Status(fiber.StatusBadRequest).JSON("Bad Request")

		}

		imageUpdate := 0
		if post.IMAGEUPDATE {
			imageUpdate = 1
		}
		img := 0
		if imageUpdate == 1 {
			if post.IMAGE == nil {
				img = 0
			} else {
				img = 1
			}
		}

		// panic(img)

		unixTimestamp := time.Now().Unix()
		_, err := db.Exec(`UPDATE POST SET author=?, message=?, likes=? , last_update=? , image= IF( ? = 1 , ?  , image ) ,
		img_last_update=IF(?  =1 , ? , img_last_update)
		WHERE uuid =?;`, post.AUTHOR, post.MESSAGES, post.LIKES, unixTimestamp, imageUpdate, img, imageUpdate, unixTimestamp, uuid)
		if err != nil {
			return c.SendStatus(404)
		}

		// post.IMAGE != nil
		if imageUpdate == 1 && img == 1 {
			imageBytes, err := base64.StdEncoding.DecodeString(*post.IMAGE)
			if err != nil {
				// handle error
				fmt.Println(err)
			}
			file, err := os.OpenFile(fmt.Sprintf("dump/%s.jpg", uuid), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			if err != nil {
				// handle error
				fmt.Println(err)

			}
			defer file.Close()
			_, err = file.Write(imageBytes)
			if err != nil {
				// handle error
				log.Fatal(err)
			}
		}

		return c.SendStatus(204)
	})

	app.Delete("/api/messages/:uuid", func(c *fiber.Ctx) error {

		uuid := c.Params("uuid")

		unixTimestamp := time.Now().Unix()
		_, err := db.Exec("UPDATE POST SET del=1, last_update=? WHERE uuid=?", unixTimestamp, uuid)
		if err != nil {
			return c.SendStatus(409)
		}
		return c.SendStatus(204)
	})

	app.Listen(":3000")
}
